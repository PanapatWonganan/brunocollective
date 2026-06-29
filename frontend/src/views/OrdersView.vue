<template>
  <div>
    <div class="d-flex flex-wrap align-center ga-3 mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Orders</div>
        <div class="text-caption text-medium-emphasis">Manage customer orders and payments</div>
      </div>
      <v-spacer />
      <div class="d-flex ga-2">
        <v-btn
          variant="tonal"
          color="secondary"
          prepend-icon="mdi-printer"
          class="text-none"
          :disabled="!printableOrders.length"
          @click="printAllLabels"
        >
          <span class="d-none d-sm-inline">Print All Labels</span>
          <span class="d-sm-none">Print</span>
          ({{ printableOrders.length }})
        </v-btn>
        <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog" class="text-none">
          New Order
        </v-btn>
      </div>
    </div>

    <!-- Status Filter Chips -->
    <div class="d-flex flex-wrap ga-2 mb-4">
      <v-chip
        :color="!statusFilter ? 'primary' : undefined"
        :variant="!statusFilter ? 'elevated' : 'outlined'"
        @click="statusFilter = ''; fetchOrders()"
        class="cursor-pointer"
      >
        All
      </v-chip>
      <v-chip
        v-for="s in statusList"
        :key="s"
        :color="!statusFilter || statusFilter === s ? statusColor(s) : undefined"
        :variant="statusFilter === s ? 'elevated' : 'outlined'"
        @click="statusFilter = s; fetchOrders()"
        class="cursor-pointer text-capitalize"
      >
        {{ s }}
      </v-chip>
    </div>

    <v-card>
      <v-card-text class="pa-5">
        <v-data-table
          :headers="headers"
          :items="orders"
          :loading="loading"
          items-per-page="10"
          class="rounded-lg"
        >
          <template v-slot:item.id="{ item }">
            <span class="font-weight-bold text-primary">#{{ item.id }}</span>
          </template>
          <template v-slot:item.customer="{ item }">
            <div class="d-flex align-center">
              <v-avatar size="30" color="primary" variant="tonal">
                <span class="text-caption font-weight-bold">{{ (item.customer?.name || '?')[0] }}</span>
              </v-avatar>
              <span class="ml-2 font-weight-medium">{{ item.customer?.name || '-' }}</span>
            </div>
          </template>
          <template v-slot:item.total_amount="{ item }">
            <span class="font-weight-bold">{{ formatCurrency(item.total_amount) }}</span>
          </template>
          <template v-slot:item.items="{ item }">
            <v-chip size="small" variant="tonal" color="primary" label>
              {{ item.items?.length || 0 }} items
            </v-chip>
          </template>
          <template v-slot:item.status="{ item }">
            <v-chip :color="statusColor(item.status)" size="small" variant="tonal" label class="text-capitalize">
              {{ item.status }}
            </v-chip>
          </template>
          <template v-slot:item.slip_image="{ item }">
            <v-btn v-if="item.slip_image" icon size="small" variant="tonal" color="success" @click="viewSlip(item)">
              <v-icon icon="mdi-image-check" size="18" />
            </v-btn>
            <v-btn v-else icon size="small" variant="tonal" color="warning" @click="openUploadDialog(item)">
              <v-icon icon="mdi-upload" size="18" />
            </v-btn>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-btn icon="mdi-eye-outline" size="small" variant="text" color="primary" @click="viewOrder(item)" />
            <v-tooltip text="Print Shipping Label" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-printer-outline" size="small" variant="text" color="secondary"
                  @click="printLabel(item)" :disabled="!item.customer?.address" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" @click="confirmDelete(item)" />
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Create Order Dialog -->
    <v-dialog v-model="createDialog" max-width="650" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">New Order</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-select
            v-model="orderForm.customer_id"
            :items="customers"
            item-title="name"
            item-value="id"
            label="Select Customer"
            prepend-inner-icon="mdi-account-outline"
            :rules="[v => !!v || 'Required']"
            class="mb-2"
          />

          <div class="d-flex align-center mb-3">
            <div class="text-subtitle-2 font-weight-medium">Order Items</div>
            <v-spacer />
            <v-btn variant="tonal" size="small" prepend-icon="mdi-plus" color="primary"
              @click="orderForm.items.push({ product_id: 0, quantity: 1 })" class="text-none">
              Add Item
            </v-btn>
          </div>

          <v-card v-for="(item, i) in orderForm.items" :key="i" variant="outlined" rounded="lg" class="mb-2">
            <v-card-text class="pa-3">
              <v-row align="center" no-gutters>
                <v-col cols="6" class="pr-2">
                  <v-select
                    v-model="item.product_id"
                    :items="products"
                    item-title="name"
                    item-value="id"
                    label="Product"
                    density="compact"
                    hide-details
                  >
                    <template v-slot:item="{ item: prod, props }">
                      <v-list-item v-bind="props" :subtitle="`Stock: ${prod.raw.stock} | ${formatCurrency(prod.raw.price)}`" />
                    </template>
                  </v-select>
                </v-col>
                <v-col cols="3" class="px-2">
                  <v-text-field v-model.number="item.quantity" label="Qty" type="number" min="1" density="compact" hide-details />
                </v-col>
                <v-col cols="3" class="pl-2 text-right">
                  <v-btn icon="mdi-close" size="small" variant="text" color="error"
                    @click="orderForm.items.splice(i, 1)"
                    :disabled="orderForm.items.length === 1" />
                </v-col>
              </v-row>
            </v-card-text>
          </v-card>

          <v-textarea v-model="orderForm.notes" label="Notes (optional)" rows="2" class="mt-3" />

          <v-file-input
            v-model="createSlipFile"
            label="Payment Slip (optional)"
            accept="image/*"
            prepend-icon="mdi-camera"
            show-size
            class="mt-2"
          />
          <v-img v-if="createSlipPreview" :src="createSlipPreview" max-height="200" contain
            class="rounded-lg border mt-2" style="background: #f8f8f8;" />
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn @click="createDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" @click="createOrder" class="text-none px-6">Create Order</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- View Order Dialog -->
    <v-dialog v-model="viewDialog" max-width="620">
      <v-card v-if="selectedOrder">
        <div class="order-header pa-5 pb-3">
          <div class="d-flex align-center">
            <div>
              <div class="text-h6 font-weight-bold">Order #{{ selectedOrder.id }}</div>
              <div class="text-caption text-medium-emphasis">{{ formatDate(selectedOrder.created_at) }}</div>
            </div>
            <v-spacer />
            <v-select
              v-model="selectedOrder.status"
              :items="statusList"
              density="compact"
              hide-details
              rounded="lg"
              style="max-width: 150px"
              @update:model-value="updateStatus(selectedOrder)"
            />
          </div>
        </div>

        <v-divider />

        <v-card-text class="pa-5">
          <div class="d-flex align-center mb-4">
            <v-avatar size="36" color="primary" variant="tonal" class="mr-3">
              <span class="text-caption font-weight-bold">{{ (selectedOrder.customer?.name || '?')[0] }}</span>
            </v-avatar>
            <div>
              <div class="font-weight-medium">{{ selectedOrder.customer?.name }}</div>
              <div class="text-caption text-medium-emphasis">Customer</div>
            </div>
            <v-spacer />
            <div class="text-right">
              <div class="text-h6 font-weight-bold text-primary">{{ formatCurrency(selectedOrder.total_amount) }}</div>
              <div class="text-caption text-medium-emphasis">Total</div>
            </div>
          </div>

          <v-table density="compact" class="rounded-lg border mb-4">
            <thead>
              <tr class="bg-grey-lighten-4">
                <th>Product</th>
                <th class="text-end">Price</th>
                <th class="text-center">Qty</th>
                <th class="text-end">Subtotal</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in selectedOrder.items" :key="item.id">
                <td class="font-weight-medium">{{ item.product?.name }}</td>
                <td class="text-end">{{ formatCurrency(item.price) }}</td>
                <td class="text-center">{{ item.quantity }}</td>
                <td class="text-end font-weight-medium">{{ formatCurrency(item.price * item.quantity) }}</td>
              </tr>
            </tbody>
          </v-table>

          <!-- Payment Slip -->
          <div v-if="selectedOrder.slip_image" class="slip-section pa-4 rounded-lg mb-3">
            <div class="text-subtitle-2 font-weight-medium mb-2">
              <v-icon icon="mdi-check-circle" color="success" size="16" class="mr-1" />
              Payment Slip
            </div>
            <v-img :src="`/uploads/${selectedOrder.slip_image}`" max-height="280" contain
              class="rounded-lg border" style="background: #f8f8f8;" />
          </div>

          <v-btn v-else variant="outlined" color="warning" prepend-icon="mdi-upload" block
            class="text-none mb-3" @click="openUploadDialog(selectedOrder)">
            Upload Payment Slip
          </v-btn>

          <div v-if="selectedOrder.notes" class="notes-section pa-3 rounded-lg">
            <div class="text-caption font-weight-medium text-medium-emphasis mb-1">NOTES</div>
            <div class="text-body-2">{{ selectedOrder.notes }}</div>
          </div>
        </v-card-text>

        <v-card-actions class="pa-5 pt-0">
          <v-btn
            v-if="selectedOrder.customer?.address"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-printer"
            class="text-none"
            @click="printLabel(selectedOrder)"
          >
            Print Shipping Label
          </v-btn>
          <v-btn
            variant="tonal"
            color="primary"
            prepend-icon="mdi-receipt-text-outline"
            class="text-none"
            @click="openReceiptDialog(selectedOrder)"
          >
            Receipt
          </v-btn>
          <v-spacer />
          <v-btn @click="viewDialog = false" variant="text" class="text-none">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Upload Slip Dialog -->
    <v-dialog v-model="uploadDialog" max-width="480">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">Upload Payment Slip</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-file-input
            v-model="slipFile"
            label="Select slip image"
            accept="image/*"
            prepend-icon="mdi-camera"
            show-size
          />
          <v-img v-if="slipPreview" :src="slipPreview" max-height="240" contain
            class="rounded-lg border mt-2" style="background: #f8f8f8;" />
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn @click="uploadDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" @click="uploadSlip" class="text-none px-6">Upload</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Slip Viewer Dialog -->
    <v-dialog v-model="slipViewDialog" max-width="500">
      <v-card>
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <span class="text-h6 font-weight-bold">Payment Slip</span>
          <v-spacer />
          <v-btn icon="mdi-close" size="small" variant="text" @click="slipViewDialog = false" />
        </v-card-title>
        <v-card-text class="px-5 pb-5">
          <v-img v-if="slipViewUrl" :src="slipViewUrl" contain class="rounded-lg" style="background: #f8f8f8;" />
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Print Shipping Label Dialog -->
    <v-dialog v-model="printDialog" max-width="550">
      <v-card>
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <span class="text-h6 font-weight-bold">Shipping Label Preview</span>
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
              <div class="label-from-detail">Tel: 0952964145</div>
            </div>
            <div class="label-divider" />
            <div class="label-section label-to">
              <div class="label-section-title">TO (Recipient)</div>
              <div class="label-to-name">{{ printingOrder?.customer?.name }}</div>
              <div v-if="printingOrder?.customer?.phone" class="label-to-phone">
                Tel: {{ printingOrder.customer.phone }}
              </div>
              <div class="label-to-address">{{ printingOrder?.customer?.address }}</div>
            </div>
            <div class="label-divider" />
            <div class="label-footer">
              <div class="label-order-id">Order #{{ printingOrder?.id }}</div>
              <div class="label-items">{{ printingOrder?.items?.length || 0 }} item(s)</div>
            </div>
          </div>

          <div class="d-flex justify-center mt-4 ga-3">
            <v-btn variant="tonal" prepend-icon="mdi-printer" color="secondary" class="text-none px-6" @click="doPrint">
              Print Label
            </v-btn>
          </div>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Receipt (ใบเสร็จรับเงิน) -->
    <v-dialog v-model="receiptDialog" max-width="520">
      <v-card>
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <span class="text-h6 font-weight-bold">Issue Receipt · ใบเสร็จรับเงิน</span>
          <v-spacer />
          <v-btn icon="mdi-close" size="small" variant="text" @click="receiptDialog = false" />
        </v-card-title>
        <v-card-text class="px-5">
          <v-alert
            type="info" variant="tonal" density="compact" class="mb-4"
            icon="mdi-information-outline"
          >
            ออกเป็น <strong>ใบเสร็จรับเงิน</strong> เท่านั้น — ร้านยังไม่ได้จดทะเบียน VAT จึงออกใบกำกับภาษีไม่ได้
          </v-alert>

          <v-alert
            v-if="existingReceipt"
            type="success" variant="tonal" density="compact" class="mb-4"
            icon="mdi-check-circle-outline"
          >
            ออกใบเสร็จแล้ว เลขที่ <strong>{{ existingReceipt.receipt_no }}</strong> — กดพิมพ์อีกครั้งจะใช้เลขเดิม
          </v-alert>
          <v-alert
            v-else
            type="info" variant="tonal" density="compact" class="mb-4"
            icon="mdi-counter"
          >
            เลขที่ใบเสร็จจะถูกออกอัตโนมัติแบบรันต่อเนื่อง (RC-ปีเดือน-ลำดับ) เมื่อกดพิมพ์
          </v-alert>

          <v-text-field
            v-model="receiptForm.buyer_name"
            label="ชื่อลูกค้า / Bill to (name)"
            density="comfortable"
            class="mb-1"
            :readonly="!!existingReceipt"
          />
          <v-textarea
            v-model="receiptForm.buyer_address"
            label="ที่อยู่ / Address"
            rows="2"
            density="comfortable"
            class="mb-1"
            :readonly="!!existingReceipt"
          />
          <v-text-field
            v-model="receiptForm.buyer_tax_id"
            label="เลขประจำตัวผู้เสียภาษี / Tax ID (ถ้ามี)"
            density="comfortable"
            hint="กรอกได้หากลูกค้าต้องการใช้เป็นหลักฐานค่าใช้จ่าย — ไม่บังคับ"
            persistent-hint
            :readonly="!!existingReceipt"
          />
        </v-card-text>
        <v-card-actions class="pa-5 pt-2">
          <v-spacer />
          <v-btn @click="receiptDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn
            color="primary" prepend-icon="mdi-printer" class="text-none px-6"
            :loading="receiptSaving"
            @click="issueAndPrintReceipt"
          >
            {{ existingReceipt ? 'Print Receipt' : 'Issue & Print' }}
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
          <div class="text-h6 font-weight-bold mb-2">Delete Order?</div>
          <div class="text-body-2 text-medium-emphasis">
            Stock will be restored for all items in this order. This cannot be undone.
          </div>
        </v-card-text>
        <v-card-actions class="justify-center pb-5">
          <v-btn @click="deleteDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="error" :loading="saving" @click="deleteOrder" class="text-none px-6">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/services/api'

const headers = [
  { title: 'ID', key: 'id', width: '80px' },
  { title: 'Customer', key: 'customer' },
  { title: 'Items', key: 'items', align: 'center' as const },
  { title: 'Total', key: 'total_amount', align: 'end' as const },
  { title: 'Status', key: 'status' },
  { title: 'Slip', key: 'slip_image', align: 'center' as const, width: '80px' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '100px' },
]

const statusList = ['pending', 'confirmed', 'shipped', 'delivered', 'cancelled']

const orders = ref<any[]>([])
const customers = ref<any[]>([])
const products = ref<any[]>([])
const statusFilter = ref('')
const loading = ref(false)
const saving = ref(false)

const createDialog = ref(false)
const viewDialog = ref(false)
const uploadDialog = ref(false)
const slipViewDialog = ref(false)
const deleteDialog = ref(false)
const printDialog = ref(false)

const selectedOrder = ref<any>(null)
const deletingOrder = ref<any>(null)
const uploadingOrder = ref<any>(null)
const printingOrder = ref<any>(null)
const labelRef = ref<HTMLElement | null>(null)

// Receipt (ใบเสร็จรับเงิน) — issued from admin only. The shop is not VAT-registered,
// so this is a plain receipt, NOT a tax invoice (ใบกำกับภาษี). The running number
// and history are persisted server-side; the dialog issues (or re-fetches) it.
const receiptDialog = ref(false)
const receiptOrder = ref<any>(null)
const receiptSaving = ref(false)
const existingReceipt = ref<any>(null)   // set when this order already has a receipt
const receiptForm = ref({
  buyer_name: '',
  buyer_address: '',
  buyer_tax_id: '',
})
const slipFile = ref<File | null>(null)
const slipViewUrl = ref('')

const createSlipFile = ref<File | null>(null)

const createSlipPreview = computed(() => {
  if (createSlipFile.value) {
    return URL.createObjectURL(createSlipFile.value)
  }
  return ''
})

const slipPreview = computed(() => {
  if (slipFile.value) {
    return URL.createObjectURL(slipFile.value)
  }
  return ''
})

const orderForm = ref({
  customer_id: 0,
  notes: '',
  items: [{ product_id: 0, quantity: 1 }]
})

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n)
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString('th-TH', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

const printableOrders = computed(() =>
  orders.value.filter((o: any) => o.customer?.address)
)

function statusColor(status: string) {
  const map: Record<string, string> = {
    pending: 'warning', confirmed: 'info', shipped: 'primary', delivered: 'success', cancelled: 'error'
  }
  return map[status] || 'grey'
}

async function fetchOrders() {
  loading.value = true
  const params: any = {}
  if (statusFilter.value) params.status = statusFilter.value
  const { data } = await api.get('/orders', { params })
  orders.value = data || []
  loading.value = false
}

async function fetchMasterData() {
  const [c, p] = await Promise.all([api.get('/customers'), api.get('/products')])
  customers.value = c.data || []
  products.value = p.data || []
}

function openCreateDialog() {
  orderForm.value = { customer_id: 0, notes: '', items: [{ product_id: 0, quantity: 1 }] }
  createSlipFile.value = null
  createDialog.value = true
}

async function createOrder() {
  saving.value = true
  try {
    const fd = new FormData()
    fd.append('customer_id', String(orderForm.value.customer_id))
    fd.append('notes', orderForm.value.notes)
    fd.append('items', JSON.stringify(orderForm.value.items))
    if (createSlipFile.value) {
      fd.append('slip', createSlipFile.value)
    }
    await api.post('/orders', fd)
    createDialog.value = false
    await Promise.all([fetchOrders(), fetchMasterData()])
  } finally {
    saving.value = false
  }
}

function viewOrder(order: any) {
  selectedOrder.value = { ...order }
  viewDialog.value = true
}

async function updateStatus(order: any) {
  await api.put(`/orders/${order.id}/status`, { status: order.status })
  await fetchOrders()
}

function openUploadDialog(order: any) {
  uploadingOrder.value = order
  slipFile.value = null
  uploadDialog.value = true
}

async function uploadSlip() {
  if (!slipFile.value) return
  saving.value = true
  try {
    const fd = new FormData()
    fd.append('slip', slipFile.value)
    await api.post(`/orders/${uploadingOrder.value.id}/slip`, fd)
    uploadDialog.value = false
    viewDialog.value = false
    await fetchOrders()
  } finally {
    saving.value = false
  }
}

function viewSlip(order: any) {
  slipViewUrl.value = `/uploads/${order.slip_image}`
  slipViewDialog.value = true
}

function printLabel(order: any) {
  printingOrder.value = order
  printDialog.value = true
}

function doPrint() {
  if (!labelRef.value) return
  const printContent = labelRef.value.innerHTML
  const win = window.open('', '_blank', 'width=500,height=600')
  if (!win) return
  win.document.write(`<!DOCTYPE html>
<html><head><title>Shipping Label - Order #${printingOrder.value?.id}</title>
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
  .label-items { font-size: 11px; color: #8C8478; }
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

// --- Receipt (ใบเสร็จรับเงิน) ---

async function openReceiptDialog(order: any) {
  receiptOrder.value = order
  existingReceipt.value = null
  receiptForm.value = {
    buyer_name: order?.customer?.name || '',
    buyer_address: order?.customer?.address || '',
    buyer_tax_id: '',
  }
  receiptDialog.value = true
  // If a receipt was already issued for this order, prefill from it (and show
  // its running number so it's clear re-printing won't allocate a new one).
  try {
    const { data } = await api.get(`/orders/${order.id}/receipt`)
    existingReceipt.value = data
    receiptForm.value = {
      buyer_name: data.buyer_name || '',
      buyer_address: data.buyer_address || '',
      buyer_tax_id: data.buyer_tax_id || '',
    }
  } catch {
    // 404 = not issued yet; that's fine.
  }
}

// Issue (or re-fetch) the persisted receipt, then print it from the saved data.
async function issueAndPrintReceipt() {
  const order = receiptOrder.value
  if (!order) return
  receiptSaving.value = true
  try {
    const { data } = await api.post(`/orders/${order.id}/receipt`, receiptForm.value)
    existingReceipt.value = data
    printReceipt(data)
  } finally {
    receiptSaving.value = false
  }
}

function esc(s: any) {
  return String(s ?? '').replace(/[&<>"']/g, (c) => (
    { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c] as string
  ))
}

// Render + print a persisted receipt using its snapshot data (lines, number,
// issue date) so the printed document always matches what was saved.
function printReceipt(receipt: any) {
  const issued = new Date(receipt.issued_at).toLocaleDateString('th-TH', {
    year: 'numeric', month: 'long', day: 'numeric',
  })

  const rows = (receipt.lines || []).map((line: any, i: number) => `
    <tr>
      <td class="c">${i + 1}</td>
      <td>${esc(line.name || '-')}${line.size ? ` <span class="muted">(${esc(line.size)})</span>` : ''}</td>
      <td class="r">${formatCurrency(line.price)}</td>
      <td class="c">${line.quantity}</td>
      <td class="r">${formatCurrency(line.price * line.quantity)}</td>
    </tr>`).join('')

  const win = window.open('', '_blank', 'width=820,height=1000')
  if (!win) return
  win.document.write(`<!DOCTYPE html>
<html lang="th"><head><meta charset="utf-8" /><title>ใบเสร็จรับเงิน ${esc(receipt.receipt_no)}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: 'Sarabun', 'Segoe UI', Tahoma, sans-serif; color: #1A1714; padding: 16mm; font-size: 13px; }
  .head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8mm; }
  .brand-name { font-size: 18px; font-weight: 700; letter-spacing: 1px; }
  .brand-detail { font-size: 11px; color: #666; margin-top: 2mm; line-height: 1.6; }
  .doc-title { text-align: right; }
  .doc-title h1 { font-size: 20px; font-weight: 700; }
  .doc-title .en { font-size: 11px; color: #8C8478; letter-spacing: 1px; text-transform: uppercase; }
  .doc-meta { font-size: 11px; color: #666; margin-top: 3mm; line-height: 1.7; }
  .parties { display: flex; gap: 8mm; margin-bottom: 6mm; }
  .party { flex: 1; background: #FAF8F5; border: 1px solid #E8E2D9; border-radius: 3mm; padding: 4mm; }
  .party .lbl { font-size: 9px; font-weight: 700; letter-spacing: 1px; color: #8C8478; text-transform: uppercase; margin-bottom: 2mm; }
  .party .nm { font-size: 14px; font-weight: 600; }
  .party .ad { font-size: 12px; color: #444; margin-top: 1mm; line-height: 1.5; white-space: pre-wrap; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 6mm; }
  th { background: #1A1714; color: #fff; font-size: 11px; font-weight: 600; padding: 2.5mm 3mm; text-align: left; }
  td { padding: 2.5mm 3mm; border-bottom: 1px solid #E8E2D9; font-size: 12px; }
  th.r, td.r { text-align: right; } th.c, td.c { text-align: center; }
  .muted { color: #8C8478; font-size: 11px; }
  .totals { display: flex; justify-content: flex-end; }
  .totals table { width: 70mm; }
  .totals td { border: none; padding: 1.5mm 3mm; }
  .totals .grand td { border-top: 2px solid #1A1714; font-size: 15px; font-weight: 700; padding-top: 3mm; }
  .note { margin-top: 8mm; font-size: 10px; color: #8C8478; line-height: 1.6; border-top: 1px dashed #ccc; padding-top: 4mm; }
  .sign { display: flex; justify-content: space-between; margin-top: 14mm; }
  .sign .box { text-align: center; font-size: 11px; color: #666; }
  .sign .line { width: 55mm; border-top: 1px solid #999; margin: 0 auto 2mm; padding-top: 10mm; }
  @page { size: A4; margin: 0; }
  @media print { body { padding: 16mm; } }
</style></head><body>
  <div class="head">
    <div>
      <div class="brand-name">Bruno Collective</div>
      <div class="brand-detail">
        87/4-5 ถนนกลางเมือง ตำบลในเมือง<br />
        อำเภอเมืองขอนแก่น ขอนแก่น 40000<br />
        โทร. 0952964145
      </div>
    </div>
    <div class="doc-title">
      <h1>ใบเสร็จรับเงิน</h1>
      <div class="en">Receipt</div>
      <div class="doc-meta">
        เลขที่ / No.: <b>${esc(receipt.receipt_no)}</b><br />
        วันที่ / Date: ${esc(issued)}<br />
        อ้างอิงออเดอร์ / Order: #${esc(receipt.order_id)}
      </div>
    </div>
  </div>

  <div class="parties">
    <div class="party">
      <div class="lbl">ลูกค้า / Bill To</div>
      <div class="nm">${esc(receipt.buyer_name || '-')}</div>
      <div class="ad">${esc(receipt.buyer_address || '-')}</div>
      ${receipt.buyer_tax_id ? `<div class="ad">เลขประจำตัวผู้เสียภาษี / Tax ID: ${esc(receipt.buyer_tax_id)}</div>` : ''}
    </div>
  </div>

  <table>
    <thead>
      <tr>
        <th class="c" style="width:10mm">#</th>
        <th>รายการ / Description</th>
        <th class="r" style="width:28mm">ราคา/หน่วย</th>
        <th class="c" style="width:16mm">จำนวน</th>
        <th class="r" style="width:30mm">รวม</th>
      </tr>
    </thead>
    <tbody>${rows}</tbody>
  </table>

  <div class="totals">
    <table>
      <tr class="grand">
        <td>ยอดรวมทั้งสิ้น / Total</td>
        <td class="r">${formatCurrency(receipt.total_amount)}</td>
      </tr>
    </table>
  </div>

  <div class="note">
    เอกสารนี้เป็น <b>ใบเสร็จรับเงิน</b> เท่านั้น มิใช่ใบกำกับภาษี — ร้านค้ายังมิได้จดทะเบียนภาษีมูลค่าเพิ่ม (VAT)<br />
    This is a receipt only, not a tax invoice. Bruno Collective is not VAT-registered.
  </div>

  <div class="sign">
    <div class="box"><div class="line"></div>ผู้รับเงิน / Received by</div>
    <div class="box"><div class="line"></div>ผู้รับสินค้า / Received by customer</div>
  </div>

  <script>window.onload=function(){window.print();window.onafterprint=function(){window.close();}}<\/script>
</body></html>`)
  win.document.close()
}

function printAllLabels() {
  const items = printableOrders.value
  if (!items.length) return

  const labelHtml = items.map((order: any) => `
    <div class="shipping-label">
      <div class="label-header">
        <img src="${window.location.origin}/brunocollective_logo.jpg" alt="Bruno Collective" class="label-logo" />
        <div class="label-brand">BRUNO COLLECTIVE</div>
      </div>
      <div class="label-divider"></div>
      <div class="label-section">
        <div class="label-section-title">FROM (Sender)</div>
        <div class="label-from-name">Bruno Collective</div>
        <div class="label-from-detail">87/4-5 ถนน กลางเมือง ตำบลในเมือง</div>
        <div class="label-from-detail">อำเภอเมืองขอนแก่น ขอนแก่น 40000</div>
        <div class="label-from-detail">Tel: 0952964145</div>
      </div>
      <div class="label-divider"></div>
      <div class="label-section label-to">
        <div class="label-section-title">TO (Recipient)</div>
        <div class="label-to-name">${order.customer?.name || ''}</div>
        ${order.customer?.phone ? `<div class="label-to-phone">Tel: ${order.customer.phone}</div>` : ''}
        <div class="label-to-address">${order.customer?.address || ''}</div>
      </div>
      <div class="label-divider"></div>
      <div class="label-footer">
        <div class="label-order-id">Order #${order.id}</div>
        <div class="label-items">${order.items?.length || 0} item(s)</div>
      </div>
    </div>
  `).join('')

  const win = window.open('', '_blank', 'width=500,height=600')
  if (!win) return
  win.document.write(`<!DOCTYPE html>
<html><head><title>Shipping Labels - ${items.length} orders</title>
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
  .label-items { font-size: 11px; color: #8C8478; }
  @media print {
    body { padding: 0; }
    .shipping-label { border: 2px solid #000; page-break-after: always; }
  }
</style></head><body>
${labelHtml}
<script>window.onload=function(){window.print();window.onafterprint=function(){window.close();}}<\/script>
</body></html>`)
  win.document.close()
}

function confirmDelete(order: any) {
  deletingOrder.value = order
  deleteDialog.value = true
}

async function deleteOrder() {
  saving.value = true
  try {
    await api.delete(`/orders/${deletingOrder.value.id}`)
    deleteDialog.value = false
    await fetchOrders()
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await Promise.all([fetchOrders(), fetchMasterData()])
})
</script>

<style scoped>
.cursor-pointer {
  cursor: pointer;
}

.order-header {
  background: linear-gradient(135deg, #FAF8F5, #F5EFE4);
}

.slip-section {
  background: #F5F8F5;
  border: 1px solid #C5D6C7;
}

.notes-section {
  background: #FAF8F5;
  border: 1px solid #E8E2D9;
}

/* Shipping Label Preview */
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

.label-items {
  font-size: 11px;
  color: #8C8478;
}
</style>
