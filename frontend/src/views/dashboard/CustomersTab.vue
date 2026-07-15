<template>
  <div>
    <!-- Headline tiles -->
    <v-row>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">ลูกค้าทั้งหมด</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">{{ data.total_customers || 0 }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">เคยซื้อแล้ว</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">{{ data.buyers || 0 }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">ซื้อซ้ำ</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">{{ data.repeat_customers || 0 }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">Repeat Rate</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #B45309;">{{ (data.repeat_rate || 0).toFixed(1) }}%</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- RFM segments -->
    <v-card class="mt-4">
      <v-card-text class="pa-5">
        <div class="mb-4">
          <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">กลุ่มลูกค้า (RFM)</div>
          <div class="text-caption" style="color: #6B7280;">
            แบ่งจากความถี่ + ความสดของการซื้อ — คลิกดูรายชื่อเพื่อตามยิงโปร/ทวงใจลูกค้า
          </div>
        </div>
        <v-row>
          <v-col v-for="seg in segmentCards" :key="seg.key" cols="12" sm="6" lg="4">
            <v-card
              variant="outlined" rounded="lg" class="segment-card"
              :style="{ borderColor: seg.color + '55' }"
              @click="openSegment(seg.key)"
            >
              <v-card-text class="pa-4">
                <div class="d-flex align-center">
                  <v-avatar size="40" :color="seg.color" variant="tonal" class="mr-3">
                    <v-icon :icon="seg.icon" size="20" />
                  </v-avatar>
                  <div>
                    <div class="font-weight-bold" style="color: #111827;">{{ seg.key }}</div>
                    <div class="text-caption" style="color: #6B7280;">{{ seg.hint }}</div>
                  </div>
                  <v-spacer />
                  <div class="text-h5 font-weight-bold" :style="{ color: seg.color }">{{ seg.count }}</div>
                </div>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <!-- New vs Returning + Top spenders -->
    <v-row class="mt-1">
      <v-col cols="12" md="6">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ลูกค้าใหม่ vs ซื้อซ้ำ</div>
              <div class="text-caption" style="color: #6B7280;">ออเดอร์รายเดือน 6 เดือนล่าสุด</div>
            </div>
            <apexchart type="bar" height="300" :options="monthlyOptions" :series="monthlySeries" />
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="6">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-3">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Top Spenders</div>
              <div class="text-caption" style="color: #6B7280;">ลูกค้าที่ใช้จ่ายสูงสุด 10 อันดับ</div>
            </div>
            <v-table v-if="topSpenders.length" density="comfortable" class="rounded-lg">
              <thead>
                <tr>
                  <th class="table-header">ลูกค้า</th>
                  <th class="table-header text-center">ออเดอร์</th>
                  <th class="table-header text-center">ซื้อล่าสุด</th>
                  <th class="table-header text-end">ยอดสะสม</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(cust, i) in topSpenders" :key="cust.id">
                  <td>
                    <div class="d-flex align-center">
                      <v-avatar size="28" color="secondary" variant="tonal" class="mr-2">
                        <span class="text-caption font-weight-bold">{{ i + 1 }}</span>
                      </v-avatar>
                      <div>
                        <div class="font-weight-medium">{{ cust.name }}</div>
                        <div class="text-caption text-medium-emphasis">{{ cust.phone || '-' }}</div>
                      </div>
                    </div>
                  </td>
                  <td class="text-center">{{ cust.orders }}</td>
                  <td class="text-center text-caption">{{ cust.recency_days }} วันก่อน</td>
                  <td class="text-end font-weight-medium">{{ formatCurrency(cust.spent) }}</td>
                </tr>
              </tbody>
            </v-table>
            <div v-else class="text-center pa-8" style="color: #6B7280;">
              <v-icon icon="mdi-account-group-outline" size="32" class="mb-2" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มีข้อมูลลูกค้า</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Segment member dialog -->
    <v-dialog v-model="segmentDialog" max-width="640">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">กลุ่ม: {{ selectedSegment }}</span>
          <span class="text-caption text-medium-emphasis ml-2">{{ segmentMembers.length }} คน</span>
        </v-card-title>
        <v-card-text class="px-5 pb-5">
          <v-table density="comfortable" class="rounded-lg">
            <thead>
              <tr>
                <th class="table-header">ลูกค้า</th>
                <th class="table-header text-center">ออเดอร์</th>
                <th class="table-header text-center">ซื้อล่าสุด</th>
                <th class="table-header text-end">ยอดสะสม</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="cust in segmentMembers" :key="cust.id">
                <td>
                  <div class="font-weight-medium">{{ cust.name }}</div>
                  <div class="text-caption text-medium-emphasis">{{ cust.phone || '-' }}</div>
                </td>
                <td class="text-center">{{ cust.orders }}</td>
                <td class="text-center text-caption">{{ cust.recency_days }} วันก่อน</td>
                <td class="text-end">{{ formatCurrency(cust.spent) }}</td>
              </tr>
            </tbody>
          </v-table>
        </v-card-text>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/services/api'

const data = ref<any>({})
const segmentDialog = ref(false)
const selectedSegment = ref('')

async function fetchCustomers() {
  try {
    const { data: res } = await api.get('/analytics/customers')
    data.value = res
  } catch {}
}
onMounted(fetchCustomers)

// Segment definitions — order matters (best → lost). Colors are identity, fixed.
const SEGMENT_META = [
  { key: 'VIP', icon: 'mdi-crown-outline', color: '#C4A24D', hint: 'ซื้อ 3+ ครั้ง และยังซื้ออยู่' },
  { key: 'ขาประจำ', icon: 'mdi-heart-outline', color: '#15803D', hint: 'ซื้อซ้ำใน 90 วัน' },
  { key: 'ลูกค้าใหม่', icon: 'mdi-account-plus-outline', color: '#2a78d6', hint: 'เพิ่งซื้อครั้งแรก' },
  { key: 'ทั่วไป', icon: 'mdi-account-outline', color: '#6B7280', hint: 'ซื้อบ้างเป็นครั้งคราว' },
  { key: 'กำลังจะหาย', icon: 'mdi-account-alert-outline', color: '#B45309', hint: 'ไม่ซื้อมา 3-6 เดือน — ควรทักไปหา' },
  { key: 'หายไปแล้ว', icon: 'mdi-account-off-outline', color: '#DC2626', hint: 'ไม่ซื้อเกิน 6 เดือน' },
]

const segments = computed<Record<string, any[]>>(() => data.value.segments || {})

const segmentCards = computed(() =>
  SEGMENT_META.map(m => ({ ...m, count: (segments.value[m.key] || []).length }))
)

const segmentMembers = computed(() => segments.value[selectedSegment.value] || [])

function openSegment(key: string) {
  if (!(segments.value[key] || []).length) return
  selectedSegment.value = key
  segmentDialog.value = true
}

const topSpenders = computed(() => data.value.top_spenders || [])

// ── New vs returning grouped bar ──
const monthly = computed(() => data.value.monthly || [])
const monthlySeries = computed(() => [
  { name: 'ลูกค้าใหม่', data: monthly.value.map((m: any) => m.new) },
  { name: 'ซื้อซ้ำ', data: monthly.value.map((m: any) => m.returning) },
])

const monthlyOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit', stacked: false },
  plotOptions: { bar: { horizontal: false, columnWidth: '55%', borderRadius: 4 } },
  colors: ['#2a78d6', '#1baf7a'],
  dataLabels: { enabled: false },
  xaxis: {
    categories: monthly.value.map((m: any) => m.month),
    labels: { style: { colors: '#6B7280', fontSize: '11px' } },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: { labels: { style: { colors: '#6B7280', fontSize: '11px' } } },
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3 },
  tooltip: { theme: 'light', y: { formatter: (val: number) => `${val} ออเดอร์` } },
  legend: { position: 'top', horizontalAlign: 'right', labels: { colors: '#6B7280' }, markers: { offsetX: -4 } },
}))

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n)
}
</script>

<style scoped>
.stat-card {
  border: 1px solid #E5E7EB !important;
}
.stat-label {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: #6B7280;
}
.segment-card {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
}
.segment-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(26, 23, 20, 0.08);
}
.table-header {
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: #6B7280 !important;
}
</style>
