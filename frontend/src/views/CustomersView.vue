<template>
  <div>
    <div class="d-flex flex-wrap align-center ga-3 mb-6">
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
                <div class="font-weight-medium">
                  {{ item.name }}
                  <v-chip v-if="item.is_member" size="x-small" color="secondary" variant="tonal" class="ml-1" prepend-icon="mdi-star">
                    Member
                  </v-chip>
                </div>
                <div v-if="item.email" class="text-caption text-medium-emphasis">{{ item.email }}</div>
              </div>
            </div>
          </template>
          <template v-slot:item.is_member="{ item }">
            <v-tooltip :text="item.is_member ? 'สมาชิก — ลด 5% ทุกออเดอร์ (กดเพื่อยกเลิก)' : 'กดเพื่อตั้งเป็นสมาชิก (ลด 5%)'" location="top">
              <template v-slot:activator="{ props }">
                <v-switch
                  v-bind="props"
                  :model-value="item.is_member"
                  color="secondary"
                  hide-details
                  density="compact"
                  @update:model-value="toggleMember(item)"
                />
              </template>
            </v-tooltip>
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
import { printLabels } from '@/utils/shippingLabel'

interface Customer {
  id?: number; name: string; email: string; phone: string; address: string; notes: string;
  is_member?: boolean; member_since?: string | null;
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
  { title: 'Member', key: 'is_member', width: '90px' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '100px' },
]

const customers = ref<Customer[]>([])
const search = ref('')
const loading = ref(false)
const dialog = ref(false)
const deleteDialog = ref(false)
const saving = ref(false)
const editingCustomer = ref<Customer | null>(null)
const deletingCustomer = ref<Customer | null>(null)
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

async function toggleMember(customer: Customer) {
  const { data } = await api.post(`/customers/${customer.id}/member`)
  // Update in place so the table doesn't jump.
  const idx = customers.value.findIndex(c => c.id === customer.id)
  if (idx !== -1) customers.value[idx] = data
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
  printLabels([{ customer }], `Address Label - ${customer.name}`)
}

onMounted(fetchCustomers)
</script>
