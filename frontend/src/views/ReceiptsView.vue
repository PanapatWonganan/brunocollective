<template>
  <div>
    <div class="d-flex align-center mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Receipts · ใบเสร็จรับเงิน</div>
        <div class="text-caption text-medium-emphasis">
          ประวัติใบเสร็จที่ออกแล้วทั้งหมด (เลขรันต่อเนื่อง) — มิใช่ใบกำกับภาษี
        </div>
      </div>
    </div>

    <v-row class="mb-4">
      <v-col cols="12" sm="6">
        <v-card class="mini-stat" border="false" style="border-left: 3px solid #C4A24D !important;">
          <v-card-text class="d-flex align-center pa-4">
            <v-icon icon="mdi-file-document-outline" color="primary" class="mr-3" />
            <div>
              <div class="text-caption text-medium-emphasis">Total Receipts</div>
              <div class="text-h6 font-weight-bold">{{ receipts.length }}</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6">
        <v-card class="mini-stat" border="false" style="border-left: 3px solid #5D7A5F !important;">
          <v-card-text class="d-flex align-center pa-4">
            <v-icon icon="mdi-cash-multiple" color="success" class="mr-3" />
            <div>
              <div class="text-caption text-medium-emphasis">Total Issued Value</div>
              <div class="text-h6 font-weight-bold">{{ formatCurrency(totalValue) }}</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-card>
      <v-card-text class="pa-5">
        <v-text-field
          v-model="search"
          prepend-inner-icon="mdi-magnify"
          placeholder="ค้นหาเลขที่ใบเสร็จ หรือ ชื่อลูกค้า..."
          clearable
          hide-details
          class="mb-4"
          style="max-width: 420px;"
        />

        <v-data-table
          :headers="headers"
          :items="filtered"
          :loading="loading"
          items-per-page="15"
          class="rounded-lg"
        >
          <template v-slot:item.issued_at="{ item }">
            {{ formatDate(item.issued_at) }}
          </template>
          <template v-slot:item.order_id="{ item }">
            #{{ item.order_id }}
          </template>
          <template v-slot:item.total_amount="{ item }">
            <span class="font-weight-medium">{{ formatCurrency(item.total_amount) }}</span>
          </template>
          <template v-slot:item.buyer_tax_id="{ item }">
            <span v-if="item.buyer_tax_id">{{ item.buyer_tax_id }}</span>
            <span v-else class="text-medium-emphasis">—</span>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-btn
              size="small" variant="tonal" color="primary" class="text-none"
              prepend-icon="mdi-printer" @click="printReceipt(item)"
            >
              พิมพ์ซ้ำ
            </v-btn>
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/services/api'

interface ReceiptLine { name: string; size: string; price: number; quantity: number }
interface Receipt {
  id: number; receipt_no: string; order_id: number;
  buyer_name: string; buyer_address: string; buyer_tax_id: string;
  lines: ReceiptLine[]; total_amount: number; issued_at: string;
}

const headers = [
  { title: 'Receipt No.', key: 'receipt_no' },
  { title: 'Issued', key: 'issued_at' },
  { title: 'Order', key: 'order_id', align: 'center' as const },
  { title: 'Customer', key: 'buyer_name' },
  { title: 'Tax ID', key: 'buyer_tax_id' },
  { title: 'Total', key: 'total_amount', align: 'end' as const },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '120px' },
]

const receipts = ref<Receipt[]>([])
const search = ref('')
const loading = ref(false)

const totalValue = computed(() =>
  receipts.value.reduce((sum, r) => sum + (r.total_amount || 0), 0)
)

const filtered = computed(() => {
  const q = (search.value || '').toLowerCase().trim()
  if (!q) return receipts.value
  return receipts.value.filter(r =>
    r.receipt_no.toLowerCase().includes(q) ||
    (r.buyer_name || '').toLowerCase().includes(q)
  )
})

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n)
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString('th-TH', { year: 'numeric', month: 'short', day: 'numeric' })
}

async function fetchReceipts() {
  loading.value = true
  try {
    const { data } = await api.get<Receipt[]>('/receipts')
    receipts.value = data || []
  } finally {
    loading.value = false
  }
}

function esc(s: any) {
  return String(s ?? '').replace(/[&<>"']/g, (c) => (
    { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c] as string
  ))
}

// Reprint a persisted receipt from its saved snapshot — identical layout to the
// Orders page receipt (kept in sync intentionally).
function printReceipt(receipt: Receipt) {
  const issued = new Date(receipt.issued_at).toLocaleDateString('th-TH', {
    year: 'numeric', month: 'long', day: 'numeric',
  })

  const rows = (receipt.lines || []).map((line, i) => `
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

onMounted(fetchReceipts)
</script>

<style scoped>
.mini-stat {
  border: 1px solid #E8E2D9 !important;
}
</style>
