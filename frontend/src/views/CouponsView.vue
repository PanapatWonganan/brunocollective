<template>
  <div>
    <div class="d-flex flex-wrap align-center ga-3 mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Coupons</div>
        <div class="text-caption text-medium-emphasis">Discount codes for orders and the storefront</div>
      </div>
      <v-spacer />
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog()" class="text-none">
        New Coupon
      </v-btn>
    </div>

    <v-card>
      <v-card-text class="pa-5">
        <v-data-table
          :headers="headers"
          :items="coupons"
          :loading="loading"
          items-per-page="10"
          class="rounded-lg"
        >
          <template v-slot:item.code="{ item }">
            <div class="font-weight-bold text-primary">{{ item.code }}</div>
            <div v-if="item.name" class="text-caption text-medium-emphasis">{{ item.name }}</div>
          </template>
          <template v-slot:item.discount="{ item }">
            <span class="font-weight-medium">{{ discountLabel(item) }}</span>
            <div v-if="item.min_order_amount > 0" class="text-caption text-medium-emphasis">
              min {{ formatCurrency(item.min_order_amount) }}
            </div>
          </template>
          <template v-slot:item.usage="{ item }">
            <span class="font-weight-medium">{{ item.used_count }}</span>
            <span class="text-medium-emphasis"> / {{ item.usage_limit > 0 ? item.usage_limit : '∞' }}</span>
            <div v-if="item.usage_limit_per_customer > 0" class="text-caption text-medium-emphasis">
              {{ item.usage_limit_per_customer }}/customer
            </div>
          </template>
          <template v-slot:item.period="{ item }">
            <div class="text-caption">
              {{ item.starts_at ? formatDate(item.starts_at) : '—' }} → {{ item.expires_at ? formatDate(item.expires_at) : '—' }}
            </div>
          </template>
          <template v-slot:item.status="{ item }">
            <v-chip :color="statusOf(item).color" size="small" variant="tonal" label>
              {{ statusOf(item).label }}
            </v-chip>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-tooltip :text="item.is_active ? 'Pause' : 'Resume'" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" :icon="item.is_active ? 'mdi-pause' : 'mdi-play'" size="small"
                  variant="text" color="secondary" @click="toggleCoupon(item)" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-pencil-outline" size="small" variant="text" color="primary" @click="openEditDialog(item)" />
            <v-tooltip text="Duplicate" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-content-copy" size="small" variant="text" @click="openCreateDialog(item)" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" @click="confirmDelete(item)" />
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Create / Edit Dialog -->
    <v-dialog v-model="formDialog" max-width="560" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">{{ editingId ? 'Edit Coupon' : 'New Coupon' }}</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-row dense>
            <v-col cols="7">
              <v-text-field
                v-model="form.code"
                label="Code"
                prepend-inner-icon="mdi-ticket-percent-outline"
                hint="Shoppers type this at checkout — stored uppercase"
                :rules="[v => !!v?.trim() || 'Required']"
                @update:model-value="form.code = form.code.toUpperCase()"
              />
            </v-col>
            <v-col cols="5">
              <v-switch v-model="form.is_active" label="Active" color="secondary" hide-details class="mt-1" />
            </v-col>
            <v-col cols="12">
              <v-text-field v-model="form.name" label="Name (internal, optional)" />
            </v-col>
            <v-col cols="6">
              <v-select
                v-model="form.type"
                :items="[{ title: 'Percent (%)', value: 'percent' }, { title: 'Fixed amount (THB)', value: 'fixed' }]"
                label="Discount Type"
              />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="form.value"
                :label="form.type === 'percent' ? 'Percent Off (1-100)' : 'Amount Off (THB)'"
                type="number" min="0"
              />
            </v-col>
            <v-col v-if="form.type === 'percent'" cols="6">
              <v-text-field
                v-model.number="form.max_discount"
                label="Max Discount (THB, 0 = no cap)"
                type="number" min="0"
              />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="form.min_order_amount"
                label="Min Order (THB, 0 = none)"
                type="number" min="0"
              />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.starts_at" label="Starts On (optional)" type="date" clearable />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.expires_at" label="Expires On (optional)" type="date" clearable />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="form.usage_limit"
                label="Total Uses (0 = unlimited)"
                type="number" min="0"
              />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="form.usage_limit_per_customer"
                label="Uses per Customer (0 = unlimited)"
                type="number" min="0"
              />
            </v-col>
            <v-col cols="12">
              <v-textarea v-model="form.description" label="Description (optional)" rows="2" />
            </v-col>
          </v-row>
          <v-alert v-if="formError" type="error" variant="tonal" density="compact">{{ formError }}</v-alert>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn @click="formDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveCoupon" class="text-none px-6">
            {{ editingId ? 'Save' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirm -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card>
        <v-card-title class="pa-5 pb-2 text-h6 font-weight-bold">Delete Coupon</v-card-title>
        <v-card-text class="px-5">
          Delete <strong>{{ deletingCoupon?.code }}</strong>? Past orders keep their discount —
          only the code itself is removed and can no longer be used.
        </v-card-text>
        <v-card-actions class="pa-5 pt-2">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" class="text-none px-6" :loading="saving" @click="deleteCoupon">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/services/api'

const headers = [
  { title: 'Code', key: 'code' },
  { title: 'Discount', key: 'discount', sortable: false },
  { title: 'Used', key: 'usage', sortable: false },
  { title: 'Period', key: 'period', sortable: false },
  { title: 'Status', key: 'status', sortable: false },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '170px' },
]

const coupons = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const formDialog = ref(false)
const deleteDialog = ref(false)
const formError = ref('')
const editingId = ref<number | null>(null)
const deletingCoupon = ref<any>(null)

const emptyForm = () => ({
  code: '',
  name: '',
  description: '',
  type: 'percent',
  value: 10,
  max_discount: 0,
  min_order_amount: 0,
  starts_at: '',
  expires_at: '',
  usage_limit: 0,
  usage_limit_per_customer: 0,
  is_active: true,
})
const form = ref(emptyForm())

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n)
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString('th-TH', { year: '2-digit', month: 'short', day: 'numeric' })
}

function discountLabel(c: any) {
  if (c.type === 'percent') {
    return `${c.value}%` + (c.max_discount > 0 ? ` (max ${formatCurrency(c.max_discount)})` : '')
  }
  return formatCurrency(c.value)
}

// Status shown in the table — mirrors the backend checks (except per-customer).
function statusOf(c: any): { label: string; color: string } {
  const now = new Date()
  if (!c.is_active) return { label: 'Paused', color: 'grey' }
  if (c.starts_at && now < new Date(c.starts_at)) return { label: 'Scheduled', color: 'info' }
  if (c.expires_at && now > new Date(c.expires_at)) return { label: 'Expired', color: 'error' }
  if (c.usage_limit > 0 && c.used_count >= c.usage_limit) return { label: 'Used up', color: 'warning' }
  return { label: 'Active', color: 'success' }
}

async function fetchCoupons() {
  loading.value = true
  try {
    const { data } = await api.get('/coupons')
    coupons.value = data || []
  } finally {
    loading.value = false
  }
}

// Date-only input → RFC3339 the Go backend can parse. Start counts from the
// beginning of the day; expiry covers the whole day (23:59:59 local time).
function toStartOfDay(d: string): string | null {
  return d ? new Date(`${d}T00:00:00`).toISOString() : null
}
function toEndOfDay(d: string): string | null {
  return d ? new Date(`${d}T23:59:59`).toISOString() : null
}
function toDateInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function openCreateDialog(copyFrom?: any) {
  editingId.value = null
  formError.value = ''
  form.value = copyFrom
    ? {
        ...emptyForm(),
        name: copyFrom.name,
        description: copyFrom.description,
        type: copyFrom.type,
        value: copyFrom.value,
        max_discount: copyFrom.max_discount,
        min_order_amount: copyFrom.min_order_amount,
        starts_at: toDateInput(copyFrom.starts_at),
        expires_at: toDateInput(copyFrom.expires_at),
        usage_limit: copyFrom.usage_limit,
        usage_limit_per_customer: copyFrom.usage_limit_per_customer,
      }
    : emptyForm()
  formDialog.value = true
}

function openEditDialog(c: any) {
  editingId.value = c.id
  formError.value = ''
  form.value = {
    code: c.code,
    name: c.name,
    description: c.description,
    type: c.type,
    value: c.value,
    max_discount: c.max_discount,
    min_order_amount: c.min_order_amount,
    starts_at: toDateInput(c.starts_at),
    expires_at: toDateInput(c.expires_at),
    usage_limit: c.usage_limit,
    usage_limit_per_customer: c.usage_limit_per_customer,
    is_active: c.is_active,
  }
  formDialog.value = true
}

async function saveCoupon() {
  saving.value = true
  formError.value = ''
  try {
    const payload = {
      ...form.value,
      code: form.value.code.trim().toUpperCase(),
      starts_at: toStartOfDay(form.value.starts_at),
      expires_at: toEndOfDay(form.value.expires_at),
    }
    if (editingId.value) {
      await api.put(`/coupons/${editingId.value}`, payload)
    } else {
      await api.post('/coupons', payload)
    }
    formDialog.value = false
    await fetchCoupons()
  } catch (err: any) {
    formError.value = err.response?.data?.error || 'Failed to save coupon'
  } finally {
    saving.value = false
  }
}

async function toggleCoupon(c: any) {
  await api.post(`/coupons/${c.id}/toggle`)
  await fetchCoupons()
}

function confirmDelete(c: any) {
  deletingCoupon.value = c
  deleteDialog.value = true
}

async function deleteCoupon() {
  saving.value = true
  try {
    await api.delete(`/coupons/${deletingCoupon.value.id}`)
    deleteDialog.value = false
    await fetchCoupons()
  } finally {
    saving.value = false
  }
}

onMounted(fetchCoupons)
</script>
