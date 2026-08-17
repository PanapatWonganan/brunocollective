<template>
  <div>
    <!-- Period selector -->
    <div class="d-flex align-center mb-4">
      <div class="text-caption" style="color: #6B7280;">
        วัดผลว่าช่องทางไหนปิดการขายเก่งสุด และความเร็วตอบแชทมีผลกับยอดแค่ไหน
      </div>
      <v-spacer />
      <v-btn-toggle v-model="days" density="compact" color="secondary" variant="outlined" mandatory>
        <v-btn :value="7" size="small" class="text-none">7 วัน</v-btn>
        <v-btn :value="30" size="small" class="text-none">30 วัน</v-btn>
        <v-btn :value="90" size="small" class="text-none">90 วัน</v-btn>
      </v-btn-toggle>
    </div>

    <!-- Headline tiles -->
    <v-row>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card">
          <v-card-text class="pa-5">
            <div class="stat-label">แชทที่คุยด้วย</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">{{ data.active_conversations || 0 }}</div>
            <div class="text-caption" style="color: #6B7280;">มีข้อความเข้าในช่วงนี้</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card">
          <v-card-text class="pa-5">
            <div class="stat-label">ปิดการขายจากแชท</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">
              {{ data.chat_orders || 0 }}
              <span class="text-body-2" style="color: #15803D;">({{ (data.chat_conversion_pct || 0).toFixed(0) }}%)</span>
            </div>
            <div class="text-caption" style="color: #6B7280;">ออเดอร์ที่สร้างจากแชท (% ของแชทที่คุย)</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card">
          <v-card-text class="pa-5">
            <div class="stat-label">ยอดขายจากแชท</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #B45309;">{{ formatCurrency(data.chat_revenue || 0) }}</div>
            <div class="text-caption" style="color: #6B7280;">เฉพาะออเดอร์ที่ปิดในแชท</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card">
          <v-card-text class="pa-5">
            <div class="stat-label">ความเร็วตอบ (มัธยฐาน)</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">{{ formatMinutes(data.median_response_min || 0) }}</div>
            <div class="text-caption" style="color: #6B7280;">จากลูกค้าทักถึงตอบครั้งแรก</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row class="mt-1">
      <!-- Revenue by channel -->
      <v-col cols="12" md="6">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ยอดขายตามช่องทาง</div>
              <div class="text-caption" style="color: #6B7280;">ทุกออเดอร์ในช่วงนี้ แยกตามที่มาของยอด</div>
            </div>
            <apexchart v-if="channels.length" type="bar" height="300" :options="channelOptions" :series="channelSeries" />
            <div v-else class="text-center pa-8" style="color: #6B7280;">
              <v-icon icon="mdi-chart-bar" size="32" class="mb-2" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มีออเดอร์ในช่วงนี้</div>
            </div>
            <v-table v-if="channels.length" density="compact" class="rounded-lg mt-2">
              <thead>
                <tr>
                  <th class="table-header">ช่องทาง</th>
                  <th class="table-header text-center">ออเดอร์</th>
                  <th class="table-header text-end">ยอดขาย</th>
                  <th class="table-header text-end">เฉลี่ย/บิล</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="ch in channels" :key="ch.channel">
                  <td>
                    <span class="channel-dot" :style="{ background: channelColor(ch.channel) }" />
                    {{ channelLabel(ch.channel) }}
                  </td>
                  <td class="text-center">{{ ch.orders }}</td>
                  <td class="text-end font-weight-medium">{{ formatCurrency(ch.revenue) }}</td>
                  <td class="text-end text-caption">{{ formatCurrency(ch.aov) }}</td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Reply speed vs conversion -->
      <v-col cols="12" md="6">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ความเร็วตอบ vs การปิดการขาย</div>
              <div class="text-caption" style="color: #6B7280;">
                แบ่งแชทตามเวลาตอบครั้งแรกโดยเฉลี่ย — % คือสัดส่วนแชทที่กลายเป็นออเดอร์
              </div>
            </div>
            <apexchart v-if="hasSpeedData" type="bar" height="300" :options="speedOptions" :series="speedSeries" />
            <div v-else class="text-center pa-8" style="color: #6B7280;">
              <v-icon icon="mdi-clock-fast" size="32" class="mb-2" color="grey-lighten-1" />
              <div class="text-body-2">ยังมีข้อมูลการตอบแชทไม่พอ</div>
            </div>
            <div v-if="hasSpeedData" class="text-caption mt-2" style="color: #6B7280;">
              <span v-for="b in responseBuckets" :key="b.key" class="mr-4">
                {{ b.label }}: {{ b.with_order }}/{{ b.conversations }} แชท ({{ formatCurrency(b.revenue) }})
              </span>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row class="mt-1">
      <!-- Time to close -->
      <v-col cols="12" md="6">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-3">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ใช้เวลาคุยนานแค่ไหนถึงปิดได้</div>
              <div class="text-caption" style="color: #6B7280;">
                จากลูกค้าทักครั้งแรก → ออเดอร์แรกของแชทนั้น
                <template v-if="ttc.total"> · มัธยฐาน <strong>{{ formatHours(ttc.median_hours) }}</strong></template>
              </div>
            </div>
            <template v-if="ttc.total">
              <div v-for="row in ttcRows" :key="row.label" class="mb-3">
                <div class="d-flex justify-space-between text-body-2 mb-1">
                  <span>{{ row.label }}</span>
                  <span class="font-weight-medium">{{ row.count }} ออเดอร์</span>
                </div>
                <v-progress-linear
                  :model-value="ttc.total ? (row.count / ttc.total) * 100 : 0"
                  height="8" rounded color="#2a78d6" bg-color="#EEF1F4"
                />
              </div>
            </template>
            <div v-else class="text-center pa-8" style="color: #6B7280;">
              <v-icon icon="mdi-timer-sand" size="32" class="mb-2" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มีออเดอร์ที่ปิดจากแชทในช่วงนี้</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Bot + pending deals -->
      <v-col cols="12" md="6">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-3">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ผู้ช่วยอัตโนมัติ & ดีลค้าง</div>
              <div class="text-caption" style="color: #6B7280;">บอทช่วยแบ่งเบาไปเท่าไหร่ และมีเงินค้างรออยู่แค่ไหน</div>
            </div>
            <v-row dense>
              <v-col cols="6" sm="3">
                <div class="mini-stat">
                  <div class="text-h6 font-weight-bold" style="color: #2a78d6;">{{ bot.rule_replies || 0 }}</div>
                  <div class="text-caption" style="color: #6B7280;">กฎตอบแชท</div>
                </div>
              </v-col>
              <v-col cols="6" sm="3">
                <div class="mini-stat">
                  <div class="text-h6 font-weight-bold" style="color: #1baf7a;">{{ bot.ai_replies || 0 }}</div>
                  <div class="text-caption" style="color: #6B7280;">AI ตอบ</div>
                </div>
              </v-col>
              <v-col cols="6" sm="3">
                <div class="mini-stat">
                  <div class="text-h6 font-weight-bold" style="color: #B45309;">{{ bot.ai_handoffs || 0 }}</div>
                  <div class="text-caption" style="color: #6B7280;">AI ส่งต่อให้คน</div>
                </div>
              </v-col>
              <v-col cols="6" sm="3">
                <div class="mini-stat">
                  <div class="text-h6 font-weight-bold" style="color: #DC2626;">{{ pending.count || 0 }}</div>
                  <div class="text-caption" style="color: #6B7280;">ดีลค้างโอน</div>
                </div>
              </v-col>
            </v-row>
            <v-alert
              v-if="pending.count"
              type="warning" variant="tonal" density="compact" class="mt-3"
            >
              มีเงินค้างรอชำระ <strong>{{ formatCurrency(pending.value) }}</strong> จาก {{ pending.count }} ออเดอร์ —
              ตามต่อได้ที่แท็บ "ดีลค้าง" ในหน้า Chats
            </v-alert>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import api from '@/services/api'

const data = ref<any>({})
const days = ref(30)

async function fetchChats() {
  try {
    const { data: res } = await api.get('/analytics/chats', { params: { days: days.value } })
    data.value = res
  } catch {}
}
onMounted(fetchChats)
watch(days, fetchChats)

// ── Channel identity — color follows the entity, never the rank ──
const CHANNEL_META: Record<string, { label: string; color: string }> = {
  line: { label: 'LINE', color: '#008300' },
  facebook: { label: 'Facebook', color: '#2a78d6' },
  instagram: { label: 'Instagram', color: '#4a3aa7' },
  tiktok: { label: 'TikTok', color: '#111827' },
  storefront: { label: 'หน้าเว็บร้าน', color: '#eda100' },
  'sale-page': { label: 'Sale Page', color: '#e34948' },
  'walk-in': { label: 'หน้าร้าน', color: '#6B7280' },
  unknown: { label: 'ไม่ระบุ', color: '#9CA3AF' },
}
function channelLabel(ch: string) { return CHANNEL_META[ch]?.label || ch }
function channelColor(ch: string) { return CHANNEL_META[ch]?.color || '#9CA3AF' }

const channels = computed<any[]>(() => data.value.channels || [])

const channelSeries = computed(() => [
  { name: 'ยอดขาย', data: channels.value.map(ch => Math.round(ch.revenue)) },
])
const channelOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit' },
  plotOptions: { bar: { horizontal: true, borderRadius: 4, distributed: true, barHeight: '60%' } },
  colors: channels.value.map(ch => channelColor(ch.channel)),
  dataLabels: { enabled: false },
  xaxis: {
    categories: channels.value.map(ch => channelLabel(ch.channel)),
    labels: {
      style: { colors: '#6B7280', fontSize: '11px' },
      formatter: (v: number) => formatShort(Number(v)),
    },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: { labels: { style: { colors: '#6B7280', fontSize: '12px' } } },
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3 },
  legend: { show: false },
  tooltip: { theme: 'light', y: { formatter: (val: number) => formatCurrency(val) } },
}))

// ── Reply speed vs conversion ──
const responseBuckets = computed<any[]>(() => data.value.response_buckets || [])
const hasSpeedData = computed(() => responseBuckets.value.some(b => b.conversations > 0))

const speedSeries = computed(() => [
  { name: 'อัตราปิดการขาย', data: responseBuckets.value.map(b => Number(b.conversion_pct.toFixed(1))) },
])
const speedOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit' },
  plotOptions: { bar: { columnWidth: '55%', borderRadius: 4 } },
  colors: ['#2a78d6'],
  dataLabels: {
    enabled: true,
    formatter: (val: number) => `${val}%`,
    style: { colors: ['#111827'], fontSize: '11px' },
    offsetY: -18,
  },
  xaxis: {
    categories: responseBuckets.value.map(b => b.label),
    labels: { style: { colors: '#6B7280', fontSize: '11px' } },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: {
    max: 100,
    labels: { style: { colors: '#6B7280', fontSize: '11px' }, formatter: (v: number) => `${v}%` },
  },
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3 },
  tooltip: {
    theme: 'light',
    y: {
      formatter: (val: number, opts: any) => {
        const b = responseBuckets.value[opts.dataPointIndex]
        return b ? `${val}% (${b.with_order}/${b.conversations} แชท)` : `${val}%`
      },
    },
  },
}))

// ── Time to close ──
const ttc = computed<any>(() => data.value.time_to_close || { total: 0, buckets: {}, median_hours: 0 })
const ttcRows = computed(() => [
  { label: 'ภายใน 1 ชั่วโมง', count: ttc.value.buckets?.within_1h || 0 },
  { label: 'ภายใน 1 วัน', count: ttc.value.buckets?.within_1d || 0 },
  { label: '1-3 วัน', count: ttc.value.buckets?.within_3d || 0 },
  { label: 'เกิน 3 วัน', count: ttc.value.buckets?.over_3d || 0 },
])

const bot = computed<any>(() => data.value.bot || {})
const pending = computed<any>(() => data.value.pending_deals || {})

// ── Formatters ──
function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB', maximumFractionDigits: 0 }).format(n || 0)
}
function formatShort(n: number) {
  if (n >= 1000) return `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k`
  return String(n)
}
function formatMinutes(min: number) {
  if (min <= 0) return '—'
  if (min < 60) return `${min.toFixed(0)} นาที`
  if (min < 1440) return `${(min / 60).toFixed(1)} ชม.`
  return `${(min / 1440).toFixed(1)} วัน`
}
function formatHours(hrs: number) {
  if (hrs < 1) return `${Math.round(hrs * 60)} นาที`
  if (hrs < 48) return `${hrs.toFixed(1)} ชม.`
  return `${(hrs / 24).toFixed(1)} วัน`
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
.table-header {
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: #6B7280 !important;
}
.channel-dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  margin-right: 6px;
}
.mini-stat {
  text-align: center;
  padding: 10px 4px;
  border: 1px solid #E5E7EB;
  border-radius: 10px;
}
</style>
