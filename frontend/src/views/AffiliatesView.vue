<template>
  <div>
    <div class="d-flex flex-wrap align-center ga-3 mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Affiliates</div>
        <div class="text-caption text-medium-emphasis">นายหน้า/ผู้แนะนำ — ลิงก์ ?ref= หรือโค้ดตอน checkout, ค่าคอมยืนยันเมื่อส่งสำเร็จ</div>
      </div>
      <v-spacer />
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog" class="text-none">
        New Affiliate
      </v-btn>
    </div>

    <v-card>
      <v-card-text class="pa-5">
        <v-data-table :headers="headers" :items="rows" :loading="loading" items-per-page="10" class="rounded-lg">
          <template v-slot:item.code="{ item }">
            <div class="font-weight-bold text-primary">{{ item.code }}</div>
            <div class="text-caption text-medium-emphasis">{{ item.name }} · {{ item.phone }}</div>
          </template>
          <template v-slot:item.commission_percent="{ item }">
            <span class="font-weight-medium">{{ item.commission_percent }}%</span>
          </template>
          <template v-slot:item.traffic="{ item }">
            <span class="font-weight-medium">{{ item.click_count }}</span> คลิก
            <div class="text-caption text-medium-emphasis">{{ item.orders_count }} ออเดอร์</div>
          </template>
          <template v-slot:item.pending="{ item }">
            <div>{{ formatCurrency(item.pending_amount) }}</div>
            <div class="text-caption text-medium-emphasis">รอส่งมอบ</div>
          </template>
          <template v-slot:item.confirmed="{ item }">
            <div :class="item.confirmed_amount > 0 ? 'font-weight-bold text-warning' : ''">
              {{ formatCurrency(item.confirmed_amount) }}
            </div>
            <div class="text-caption text-medium-emphasis">ค้างจ่าย</div>
          </template>
          <template v-slot:item.paid="{ item }">
            <div class="text-success">{{ formatCurrency(item.paid_amount) }}</div>
            <div class="text-caption text-medium-emphasis">จ่ายแล้ว</div>
          </template>
          <template v-slot:item.status="{ item }">
            <v-chip :color="item.is_active ? 'success' : 'grey'" size="small" variant="tonal" label>
              {{ item.is_active ? 'Active' : 'Paused' }}
            </v-chip>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-tooltip text="รายงาน + จ่ายค่าคอม" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-chart-box-outline" size="small" variant="text" color="secondary" @click="openReport(item)" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-pencil-outline" size="small" variant="text" color="primary" @click="openEditDialog(item)" />
            <v-tooltip :text="item.is_active ? 'Pause' : 'Resume'" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" :icon="item.is_active ? 'mdi-pause' : 'mdi-play'" size="small" variant="text" @click="toggleAffiliate(item)" />
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
          <span class="text-h6 font-weight-bold">{{ editingId ? 'Edit Affiliate' : 'New Affiliate' }}</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-row dense>
            <v-col cols="7">
              <v-text-field
                v-model="form.code" label="Referral Code"
                prepend-inner-icon="mdi-account-cash-outline"
                hint="ใช้ในลิงก์ ?ref= และพิมพ์ตอน checkout — เก็บเป็นตัวพิมพ์ใหญ่"
                :rules="[(v: string) => !!v?.trim() || 'Required']"
                @update:model-value="form.code = form.code.toUpperCase()"
              />
            </v-col>
            <v-col cols="5">
              <v-switch v-model="form.is_active" label="Active" color="secondary" hide-details class="mt-1" />
            </v-col>
            <v-col cols="12">
              <v-text-field v-model="form.name" label="ชื่อ (Name)" :rules="[(v: string) => !!v?.trim() || 'Required']" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.phone" label="เบอร์โทร (ใช้ login portal)" :rules="[(v: string) => !!v?.trim() || 'Required']" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.email" label="Email (optional)" />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="form.commission_percent" label="ค่าคอมมาตรฐาน (%)"
                type="number" min="0" max="100" suffix="%"
                hint="สินค้าที่ตั้ง % เฉพาะตัวจะใช้ค่านั้นแทน" persistent-hint
              />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model="form.password" :label="editingId ? 'รหัสผ่านใหม่ (เว้นว่าง = ไม่เปลี่ยน)' : 'รหัสผ่าน portal *'"
                type="password" hint="อย่างน้อย 6 ตัวอักษร" persistent-hint
              />
            </v-col>
            <v-col cols="12">
              <v-textarea v-model="form.notes" label="Notes (internal, optional)" rows="2" />
            </v-col>
          </v-row>
          <v-alert v-if="formError" type="error" variant="tonal" density="compact">{{ formError }}</v-alert>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn @click="formDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveAffiliate" class="text-none px-6">
            {{ editingId ? 'Save' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Report Dialog -->
    <v-dialog v-model="reportDialog" max-width="860">
      <v-card v-if="report">
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <div>
            <span class="text-h6 font-weight-bold">{{ report.affiliate.code }}</span>
            <span class="text-body-2 text-medium-emphasis ml-2">{{ report.affiliate.name }} · {{ report.affiliate.commission_percent }}%</span>
          </div>
          <v-spacer />
          <v-btn icon="mdi-close" variant="text" size="small" @click="reportDialog = false" />
        </v-card-title>
        <v-card-text class="px-5">
          <!-- Share link -->
          <v-alert variant="tonal" color="secondary" density="comfortable" class="mb-4">
            <div class="d-flex align-center ga-2 flex-wrap">
              <span class="text-body-2">ลิงก์แชร์:</span>
              <code class="text-body-2">{{ shareLink(report.affiliate.code) }}</code>
              <v-btn size="x-small" variant="tonal" class="text-none" prepend-icon="mdi-content-copy" @click="copyLink(report.affiliate.code)">
                {{ copied ? 'Copied!' : 'Copy' }}
              </v-btn>
            </div>
          </v-alert>

          <!-- Totals -->
          <v-row dense class="mb-4">
            <v-col cols="6" sm="3" v-for="card in reportCards" :key="card.label">
              <v-card variant="tonal" :color="card.color" class="pa-3">
                <div class="text-caption">{{ card.label }}</div>
                <div class="text-h6 font-weight-bold">{{ formatCurrency(card.value) }}</div>
              </v-card>
            </v-col>
          </v-row>

          <!-- Pay button -->
          <div v-if="report.confirmed_amount > 0" class="mb-4">
            <v-btn color="primary" class="text-none" prepend-icon="mdi-cash-check" :loading="paying" @click="payDialog = true">
              จ่ายค่าคอมมิชชั่น {{ formatCurrency(report.confirmed_amount) }}
            </v-btn>
          </div>

          <!-- Orders -->
          <v-table density="comfortable" v-if="report.orders?.length">
            <thead>
              <tr>
                <th>Order</th><th>วันที่</th><th>ลูกค้า</th><th>สถานะออเดอร์</th>
                <th class="text-right">ยอดออเดอร์</th><th class="text-right">ค่าคอม</th><th>สถานะค่าคอม</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="o in report.orders" :key="`${o.order_id}-${o.commission_status}`">
                <td>#{{ o.order_id }}</td>
                <td class="text-caption">{{ formatDate(o.created_at) }}</td>
                <td>{{ o.customer_name }}</td>
                <td><v-chip size="x-small" variant="tonal" :color="orderStatusColor(o.order_status)" label>{{ o.order_status }}</v-chip></td>
                <td class="text-right">{{ formatCurrency(o.order_total) }}</td>
                <td class="text-right font-weight-medium">{{ formatCurrency(o.commission) }}</td>
                <td><v-chip size="x-small" variant="tonal" :color="commissionColor(o.commission_status)" label>{{ commissionLabel(o.commission_status) }}</v-chip></td>
              </tr>
            </tbody>
          </v-table>
          <div v-else class="text-medium-emphasis text-body-2 py-4">ยังไม่มีออเดอร์จาก affiliate คนนี้</div>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Pay Confirm -->
    <v-dialog v-model="payDialog" max-width="420">
      <v-card>
        <v-card-title class="pa-5 pb-2 text-h6 font-weight-bold">จ่ายค่าคอมมิชชั่น</v-card-title>
        <v-card-text class="px-5">
          ยืนยันการจ่ายค่าคอมที่ยืนยันแล้วทั้งหมด <strong>{{ formatCurrency(report?.confirmed_amount || 0) }}</strong>
          ให้ <strong>{{ report?.affiliate?.name }}</strong>?
          รายการจะถูกทำเครื่องหมาย "จ่ายแล้ว" (บันทึกถาวร)
        </v-card-text>
        <v-card-actions class="pa-5 pt-2">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="payDialog = false">Cancel</v-btn>
          <v-btn color="primary" variant="flat" class="text-none px-6" :loading="paying" @click="payAffiliate">ยืนยันจ่าย</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirm -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card>
        <v-card-title class="pa-5 pb-2 text-h6 font-weight-bold">Delete Affiliate</v-card-title>
        <v-card-text class="px-5">
          ลบ <strong>{{ deleting?.code }}</strong>? ออเดอร์เก่ายังเก็บโค้ดและประวัติค่าคอมไว้ —
          แต่โค้ดนี้จะใช้ไม่ได้อีกและ affiliate จะเข้า portal ไม่ได้
        </v-card-text>
        <v-card-actions class="pa-5 pt-2">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" class="text-none px-6" :loading="saving" @click="deleteAffiliate">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="2500">{{ snackbar.text }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/services/api'

const headers = [
  { title: 'Code', key: 'code' },
  { title: '%', key: 'commission_percent', sortable: false },
  { title: 'Traffic', key: 'traffic', sortable: false },
  { title: 'รอส่งมอบ', key: 'pending', sortable: false },
  { title: 'ค้างจ่าย', key: 'confirmed', sortable: false },
  { title: 'จ่ายแล้ว', key: 'paid', sortable: false },
  { title: 'Status', key: 'status', sortable: false },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '180px' },
]

const rows = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const paying = ref(false)
const formDialog = ref(false)
const reportDialog = ref(false)
const payDialog = ref(false)
const deleteDialog = ref(false)
const formError = ref('')
const editingId = ref<number | null>(null)
const deleting = ref<any>(null)
const report = ref<any>(null)
const copied = ref(false)
const snackbar = ref({ show: false, text: '', color: 'success' })

const emptyForm = () => ({
  code: '', name: '', phone: '', email: '', password: '',
  commission_percent: 10, notes: '', is_active: true,
})
const form = ref(emptyForm())

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n || 0)
}
function formatDate(d: string) {
  return new Date(d).toLocaleDateString('th-TH', { year: '2-digit', month: 'short', day: 'numeric' })
}
function shareLink(code: string) {
  return `${location.origin}/?ref=${code}`
}
async function copyLink(code: string) {
  await navigator.clipboard.writeText(shareLink(code))
  copied.value = true
  setTimeout(() => (copied.value = false), 1800)
}
function orderStatusColor(s: string) {
  return { pending: 'warning', confirmed: 'info', shipped: 'secondary', delivered: 'success', cancelled: 'error' }[s] || 'grey'
}
function commissionColor(s: string) {
  return { pending: 'warning', confirmed: 'info', paid: 'success', cancelled: 'error' }[s] || 'grey'
}
function commissionLabel(s: string) {
  return { pending: 'รอส่งมอบ', confirmed: 'ค้างจ่าย', paid: 'จ่ายแล้ว', cancelled: 'ยกเลิก' }[s] || s
}

const reportCards = computed(() => report.value ? [
  { label: 'รอส่งมอบ', value: report.value.pending_amount, color: 'warning' },
  { label: 'ค้างจ่าย (ยืนยันแล้ว)', value: report.value.confirmed_amount, color: 'info' },
  { label: 'จ่ายแล้ว', value: report.value.paid_amount, color: 'success' },
  { label: 'ยกเลิก', value: report.value.cancelled_amount, color: 'error' },
] : [])

async function fetchAffiliates() {
  loading.value = true
  try {
    const { data } = await api.get('/affiliates')
    // Flatten {affiliate: {...}, pending_amount, ...} rows for the table.
    rows.value = (data || []).map((r: any) => ({ ...r.affiliate, ...r }))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingId.value = null
  formError.value = ''
  form.value = emptyForm()
  formDialog.value = true
}

function openEditDialog(a: any) {
  editingId.value = a.id
  formError.value = ''
  form.value = {
    code: a.code, name: a.name, phone: a.phone, email: a.email || '',
    password: '', commission_percent: a.commission_percent,
    notes: a.notes || '', is_active: a.is_active,
  }
  formDialog.value = true
}

async function saveAffiliate() {
  saving.value = true
  formError.value = ''
  try {
    const payload = { ...form.value, code: form.value.code.trim().toUpperCase() }
    if (editingId.value) {
      await api.put(`/affiliates/${editingId.value}`, payload)
    } else {
      await api.post('/affiliates', payload)
    }
    formDialog.value = false
    await fetchAffiliates()
  } catch (err: any) {
    formError.value = err.response?.data?.error || 'Failed to save affiliate'
  } finally {
    saving.value = false
  }
}

async function toggleAffiliate(a: any) {
  await api.post(`/affiliates/${a.id}/toggle`)
  await fetchAffiliates()
}

async function openReport(a: any) {
  report.value = null
  reportDialog.value = true
  const { data } = await api.get(`/affiliates/${a.id}/report`)
  report.value = data
}

async function payAffiliate() {
  if (!report.value) return
  paying.value = true
  try {
    const { data } = await api.post(`/affiliates/${report.value.affiliate.id}/pay`)
    payDialog.value = false
    snackbar.value = { show: true, text: `จ่ายแล้ว ${formatCurrency(data.amount)} (${data.count} รายการ)`, color: 'success' }
    const { data: fresh } = await api.get(`/affiliates/${report.value.affiliate.id}/report`)
    report.value = fresh
    await fetchAffiliates()
  } catch (err: any) {
    snackbar.value = { show: true, text: err.response?.data?.error || 'จ่ายไม่สำเร็จ', color: 'error' }
  } finally {
    paying.value = false
  }
}

function confirmDelete(a: any) {
  deleting.value = a
  deleteDialog.value = true
}

async function deleteAffiliate() {
  saving.value = true
  try {
    await api.delete(`/affiliates/${deleting.value.id}`)
    deleteDialog.value = false
    await fetchAffiliates()
  } finally {
    saving.value = false
  }
}

onMounted(fetchAffiliates)
</script>
