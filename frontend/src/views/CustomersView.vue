<template>
  <div>
    <div class="d-flex align-center mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Customers</div>
        <div class="text-caption text-medium-emphasis">Manage your customer records</div>
      </div>
      <v-spacer />
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openDialog()" class="text-none">
        Add Customer
      </v-btn>
    </div>

    <v-card>
      <v-card-text class="pa-5">
        <v-text-field
          v-model="search"
          prepend-inner-icon="mdi-magnify"
          placeholder="Search by name, phone, or email..."
          clearable
          hide-details
          @update:model-value="fetchCustomers"
          class="mb-4"
          style="max-width: 400px;"
        />

        <v-data-table
          :headers="headers"
          :items="customers"
          :loading="loading"
          items-per-page="10"
          class="rounded-lg"
        >
          <template v-slot:item.name="{ item }">
            <div class="d-flex align-center py-2">
              <v-avatar size="36" :color="avatarColor(item.name)" variant="tonal">
                <span class="text-caption font-weight-bold">{{ item.name[0]?.toUpperCase() }}</span>
              </v-avatar>
              <div class="ml-3">
                <div class="font-weight-medium">{{ item.name }}</div>
                <div v-if="item.email" class="text-caption text-medium-emphasis">{{ item.email }}</div>
              </div>
            </div>
          </template>
          <template v-slot:item.phone="{ item }">
            <div v-if="item.phone" class="d-flex align-center">
              <v-icon icon="mdi-phone-outline" size="14" class="mr-1 text-medium-emphasis" />
              {{ item.phone }}
            </div>
            <span v-else class="text-medium-emphasis">-</span>
          </template>
          <template v-slot:item.address="{ item }">
            <div class="text-truncate" style="max-width: 250px;">
              {{ item.address || '-' }}
            </div>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-btn icon="mdi-pencil-outline" size="small" variant="text" color="primary" @click="openDialog(item)" />
            <v-tooltip text="Print Address Label" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-printer-outline" size="small" variant="text" color="secondary"
                  @click="printLabel(item)" :disabled="!item.address" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" @click="confirmDelete(item)" />
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Create/Edit Dialog -->
    <v-dialog v-model="dialog" max-width="520" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">{{ editingCustomer ? 'Edit Customer' : 'New Customer' }}</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-form ref="form">
            <v-text-field v-model="formData.name" label="Full Name" prepend-inner-icon="mdi-account-outline" :rules="[v => !!v || 'Required']" class="mb-1" />
            <v-text-field v-model="formData.email" label="Email" prepend-inner-icon="mdi-email-outline" class="mb-1" />
            <v-text-field v-model="formData.phone" label="Phone" prepend-inner-icon="mdi-phone-outline" class="mb-1" />
            <v-textarea v-model="formData.address" label="Address" prepend-inner-icon="mdi-map-marker-outline" rows="2" class="mb-1" />
            <v-textarea v-model="formData.notes" label="Notes" prepend-inner-icon="mdi-note-outline" rows="2" />
          </v-form>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn @click="dialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveCustomer" class="text-none px-6">
            {{ editingCustomer ? 'Update' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Print Address Label Dialog -->
    <v-dialog v-model="printDialog" max-width="550">
      <v-card>
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <span class="text-h6 font-weight-bold">Address Label Preview</span>
          <v-spacer />
          <v-btn icon="mdi-close" size="small" variant="text" @click="printDialog = false" />
        </v-card-title>
        <v-card-text class="px-5 pb-5">
          <div ref="labelRef" class="shipping-label">
            <div class="label-header">
              <img src="/brunocollective_logo.jpg" alt="Bruno Collective" class="label-logo" />
              <div class="label-brand">BRUNO COLLECTIVE</div>
            </div>
            <div class="label-divider" />
            <div class="label-section">
              <div class="label-section-title">FROM (Sender)</div>
              <div class="label-from-name">Bruno Collective</div>
              <div class="label-from-detail">87/4-5 ถนน กลางเมือง ตำบลในเมือง</div>
              <div class="label-from-detail">อำเภอเมืองขอนแก่น ขอนแก่น 40000</div>
              <div class="label-from-detail">Tel: 081-4469442</div>
            </div>
            <div class="label-divider" />
            <div class="label-section label-to">
              <div class="label-section-title">TO (Recipient)</div>
              <div class="label-to-name">{{ printingCustomer?.name }}</div>
              <div v-if="printingCustomer?.phone" class="label-to-phone">
                Tel: {{ printingCustomer.phone }}
              </div>
              <div class="label-to-address">{{ printingCustomer?.address }}</div>
            </div>
            <div class="label-divider" />
            <div class="label-footer">
              <div class="label-order-id">{{ printingCustomer?.name }}</div>
            </div>
          </div>

          <div class="d-flex justify-center mt-4">
            <v-btn variant="tonal" prepend-icon="mdi-printer" color="secondary" class="text-none px-6" @click="doPrint">
              Print Label
            </v-btn>
          </div>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Delete Confirm -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card class="text-center pa-2">
        <v-card-text class="pt-5">
          <v-avatar color="error" variant="tonal" size="56" class="mb-4">
            <v-icon icon="mdi-delete-outline" size="28" />
          </v-avatar>
          <div class="text-h6 font-weight-bold mb-2">Delete Customer?</div>
          <div class="text-body-2 text-medium-emphasis">
            Are you sure you want to delete "<strong>{{ deletingCustomer?.name }}</strong>"?
          </div>
        </v-card-text>
        <v-card-actions class="justify-center pb-5">
          <v-btn @click="deleteDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="error" :loading="saving" @click="deleteCustomer" class="text-none px-6">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/services/api'

interface Customer {
  id?: number; name: string; email: string; phone: string; address: string; notes: string;
}

const colors = ['primary', 'secondary', 'success', 'warning', 'info', 'error']

function avatarColor(name: string) {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return colors[Math.abs(hash) % colors.length]
}

const headers = [
  { title: 'Customer', key: 'name' },
  { title: 'Phone', key: 'phone' },
  { title: 'Address', key: 'address' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '100px' },
]

const customers = ref<Customer[]>([])
const search = ref('')
const loading = ref(false)
const dialog = ref(false)
const deleteDialog = ref(false)
const printDialog = ref(false)
const saving = ref(false)
const editingCustomer = ref<Customer | null>(null)
const deletingCustomer = ref<Customer | null>(null)
const printingCustomer = ref<Customer | null>(null)
const labelRef = ref<HTMLElement | null>(null)
const form = ref()

const emptyForm = (): Customer => ({ name: '', email: '', phone: '', address: '', notes: '' })
const formData = ref<Customer>(emptyForm())

async function fetchCustomers() {
  loading.value = true
  const { data } = await api.get('/customers', { params: { search: search.value } })
  customers.value = data || []
  loading.value = false
}

function openDialog(customer?: Customer) {
  editingCustomer.value = customer || null
  formData.value = customer ? { ...customer } : emptyForm()
  dialog.value = true
}

async function saveCustomer() {
  saving.value = true
  try {
    if (editingCustomer.value) {
      await api.put(`/customers/${editingCustomer.value.id}`, formData.value)
    } else {
      await api.post('/customers', formData.value)
    }
    dialog.value = false
    await fetchCustomers()
  } finally {
    saving.value = false
  }
}

function confirmDelete(customer: Customer) {
  deletingCustomer.value = customer
  deleteDialog.value = true
}

async function deleteCustomer() {
  saving.value = true
  try {
    await api.delete(`/customers/${deletingCustomer.value?.id}`)
    deleteDialog.value = false
    await fetchCustomers()
  } finally {
    saving.value = false
  }
}

function printLabel(customer: Customer) {
  printingCustomer.value = customer
  printDialog.value = true
}

function doPrint() {
  if (!labelRef.value) return
  const printContent = labelRef.value.innerHTML
  const win = window.open('', '_blank', 'width=500,height=600')
  if (!win) return
  win.document.write(`<!DOCTYPE html>
<html><head><title>Address Label - ${printingCustomer.value?.name}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; padding: 0; }
  @page { size: 100mm 150mm; margin: 0; }
  .shipping-label {
    width: 100mm; min-height: 140mm; padding: 6mm;
    border: 2px solid #1A1714; margin: 0 auto;
  }
  .label-header { text-align: center; padding: 4mm 0; }
  .label-logo { height: 32px; margin-bottom: 2mm; }
  .label-brand { font-size: 10px; font-weight: 700; letter-spacing: 2px; color: #1A1714; }
  .label-divider { border-top: 1px dashed #ccc; margin: 3mm 0; }
  .label-section { padding: 2mm 0; }
  .label-section-title {
    font-size: 9px; font-weight: 700; letter-spacing: 1.5px;
    color: #8C8478; margin-bottom: 2mm; text-transform: uppercase;
  }
  .label-from-name { font-size: 13px; font-weight: 600; color: #1A1714; }
  .label-from-detail { font-size: 11px; color: #666; margin-top: 1mm; }
  .label-to { background: #FAF8F5; padding: 4mm; border-radius: 3mm; border: 1px solid #E8E2D9; }
  .label-to-name { font-size: 18px; font-weight: 700; color: #1A1714; margin-bottom: 1mm; }
  .label-to-phone { font-size: 13px; color: #555; margin-bottom: 2mm; }
  .label-to-address { font-size: 14px; line-height: 1.5; color: #333; white-space: pre-wrap; }
  .label-footer { display: flex; justify-content: space-between; align-items: center; padding: 2mm 0; }
  .label-order-id { font-size: 12px; font-weight: 700; color: #1A1714; }
  @media print {
    body { padding: 0; }
    .shipping-label { border: 2px solid #000; page-break-after: always; }
  }
</style></head><body>
<div class="shipping-label">${printContent}</div>
<script>window.onload=function(){window.print();window.onafterprint=function(){window.close();}}<\/script>
</body></html>`)
  win.document.close()
}

onMounted(fetchCustomers)
</script>

<style scoped>
.shipping-label {
  border: 2px solid #1A1714;
  border-radius: 8px;
  padding: 20px;
  background: #fff;
  max-width: 380px;
  margin: 0 auto;
}

.label-header {
  text-align: center;
  padding: 8px 0;
}

.label-logo {
  height: 32px;
  margin-bottom: 4px;
}

.label-brand {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #1A1714;
}

.label-divider {
  border-top: 1px dashed #ccc;
  margin: 10px 0;
}

.label-section {
  padding: 4px 0;
}

.label-section-title {
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 1.5px;
  color: #8C8478;
  margin-bottom: 6px;
  text-transform: uppercase;
}

.label-from-name {
  font-size: 13px;
  font-weight: 600;
  color: #1A1714;
}

.label-from-detail {
  font-size: 11px;
  color: #666;
  margin-top: 2px;
}

.label-to {
  background: #FAF8F5;
  padding: 14px;
  border-radius: 8px;
  border: 1px solid #E8E2D9;
}

.label-to-name {
  font-size: 18px;
  font-weight: 700;
  color: #1A1714;
  margin-bottom: 2px;
}

.label-to-phone {
  font-size: 13px;
  color: #555;
  margin-bottom: 6px;
}

.label-to-address {
  font-size: 14px;
  line-height: 1.5;
  color: #333;
  white-space: pre-wrap;
}

.label-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.label-order-id {
  font-size: 12px;
  font-weight: 700;
  color: #1A1714;
}
</style>
