<template>
  <div>
    <!-- Stat Cards (lifetime) -->
    <v-row>
      <v-col v-for="card in statCards" :key="card.title" cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="d-flex justify-space-between align-start">
              <div>
                <div class="stat-label">{{ card.title }}</div>
                <div class="text-h4 font-weight-bold mt-2" style="color: #111827;">{{ card.value }}</div>
              </div>
              <div class="stat-icon" :style="{ background: card.iconBg }">
                <v-icon :icon="card.icon" size="22" :color="card.iconColor" />
              </div>
            </div>
            <div v-if="card.subtitle" class="text-caption mt-3" style="color: #6B7280;">{{ card.subtitle }}</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Period KPIs with previous-period comparison -->
    <v-card class="mt-4">
      <v-card-text class="pa-5">
        <div class="d-flex align-center mb-4 flex-wrap ga-2">
          <div>
            <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ผลประกอบการช่วงนี้</div>
            <div class="text-caption" style="color: #6B7280;">เทียบกับช่วงก่อนหน้า ({{ overview.days || days }} วัน)</div>
          </div>
          <v-spacer />
          <v-btn-toggle v-model="days" mandatory density="compact" color="secondary" variant="outlined" class="period-toggle">
            <v-btn value="7" size="small" class="text-none">7 วัน</v-btn>
            <v-btn value="30" size="small" class="text-none">30 วัน</v-btn>
            <v-btn value="90" size="small" class="text-none">90 วัน</v-btn>
            <v-btn value="365" size="small" class="text-none">1 ปี</v-btn>
          </v-btn-toggle>
        </div>
        <v-row>
          <v-col v-for="kpi in kpiCards" :key="kpi.title" cols="12" sm="6" lg="3">
            <div class="kpi-tile pa-4">
              <div class="stat-label">{{ kpi.title }}</div>
              <div class="text-h5 font-weight-bold mt-1" style="color: #111827;">{{ kpi.value }}</div>
              <div class="d-flex align-center mt-1 ga-2">
                <v-chip v-if="kpi.delta !== null" size="x-small" variant="tonal" label
                  :color="kpi.delta >= 0 ? 'success' : 'error'"
                  :prepend-icon="kpi.delta >= 0 ? 'mdi-arrow-up' : 'mdi-arrow-down'">
                  {{ Math.abs(kpi.delta).toFixed(1) }}%
                </v-chip>
                <span class="text-caption" style="color: #6B7280;">{{ kpi.caption }}</span>
              </div>
            </div>
          </v-col>
        </v-row>
        <v-alert
          v-if="overview.current && overview.current.cost_coverage < 100 && overview.current.units_sold > 0"
          type="info" variant="tonal" density="compact" class="mt-3"
        >
          มีสินค้าที่ยังไม่ได้กรอกต้นทุน {{ (100 - overview.current.cost_coverage).toFixed(0) }}% ของยอดขาย —
          กำไรขั้นต้นที่แสดงจึงสูงกว่าจริง กรอกต้นทุนในหน้า Products เพื่อให้ตัวเลขแม่นยำ
        </v-alert>
      </v-card-text>
    </v-card>

    <!-- Revenue Chart -->
    <v-row class="mt-1">
      <v-col cols="12">
        <v-card>
          <v-card-text class="pa-5">
            <div class="d-flex align-center mb-4 flex-wrap ga-2">
              <div>
                <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Revenue Overview</div>
                <div class="text-caption" style="color: #6B7280;">Sales performance over time</div>
              </div>
              <v-spacer />
              <v-btn-toggle v-model="selectedPeriod" mandatory density="compact" color="secondary" variant="outlined" class="period-toggle">
                <v-btn value="day" size="small" class="text-none">Day</v-btn>
                <v-btn value="week" size="small" class="text-none">Week</v-btn>
                <v-btn value="month" size="small" class="text-none">Month</v-btn>
                <v-btn value="year" size="small" class="text-none">Year</v-btn>
              </v-btn-toggle>
            </div>
            <apexchart
              type="area"
              height="320"
              :options="revenueChartOptions"
              :series="revenueChartSeries"
            />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Channel + Category -->
    <v-row class="mt-1">
      <v-col cols="12" md="5">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ยอดขายตามช่องทาง</div>
              <div class="text-caption" style="color: #6B7280;">ช่วง {{ overview.days || days }} วันที่ผ่านมา</div>
            </div>
            <apexchart v-if="channels.length" type="donut" height="280" :options="channelOptions" :series="channelSeries" />
            <div v-else class="text-center pa-12" style="color: #6B7280;">
              <v-icon icon="mdi-storefront-outline" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มีข้อมูลช่องทาง — เลือกช่องทางตอนสร้างออเดอร์</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="7">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ยอดขายตามหมวดสินค้า</div>
              <div class="text-caption" style="color: #6B7280;">ช่วง {{ overview.days || days }} วันที่ผ่านมา</div>
            </div>
            <apexchart v-if="categories.length" type="bar" height="280" :options="categoryOptions" :series="categorySeries" />
            <div v-else class="text-center pa-12" style="color: #6B7280;">
              <v-icon icon="mdi-hanger" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มีข้อมูล — กำหนดหมวดสินค้าในหน้า Products</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Order Status + Top Products -->
    <v-row class="mt-1">
      <v-col cols="12" md="5">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Order Status</div>
              <div class="text-caption" style="color: #6B7280;">Distribution by status</div>
            </div>
            <apexchart
              type="donut"
              height="280"
              :options="orderStatusOptions"
              :series="orderStatusSeries"
            />
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="7">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Top Selling Products</div>
              <div class="text-caption" style="color: #6B7280;">By quantity sold</div>
            </div>
            <apexchart
              v-if="topSellingData.length"
              type="bar"
              height="280"
              :options="topSellingOptions"
              :series="topSellingSeries"
            />
            <div v-else class="text-center pa-12" style="color: #6B7280;">
              <v-icon icon="mdi-chart-bar" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">No sales data yet</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Coupons + Sale Pages performance -->
    <v-row class="mt-1">
      <v-col cols="12" lg="5">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-3">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ประสิทธิภาพคูปอง</div>
              <div class="text-caption" style="color: #6B7280;">ช่วง {{ overview.days || days }} วันที่ผ่านมา</div>
            </div>
            <v-table v-if="coupons.length" density="comfortable" class="rounded-lg">
              <thead>
                <tr>
                  <th class="table-header">โค้ด</th>
                  <th class="table-header text-center">ใช้ไป</th>
                  <th class="table-header text-end">ยอดขายที่สร้าง</th>
                  <th class="table-header text-end">ส่วนลดที่ให้</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="cp in coupons" :key="cp.code">
                  <td class="font-weight-medium">{{ cp.code }}</td>
                  <td class="text-center">{{ cp.uses }}</td>
                  <td class="text-end">{{ formatCurrency(cp.revenue) }}</td>
                  <td class="text-end" style="color: #DC2626;">-{{ formatCurrency(cp.discount) }}</td>
                </tr>
              </tbody>
            </v-table>
            <div v-else class="text-center pa-8" style="color: #6B7280;">
              <v-icon icon="mdi-ticket-percent-outline" size="32" class="mb-2" color="grey-lighten-1" />
              <div class="text-body-2">ไม่มีออเดอร์ที่ใช้คูปองในช่วงนี้</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" lg="7">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-3">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Sale Pages — Conversion</div>
              <div class="text-caption" style="color: #6B7280;">สถิติสะสมของแต่ละเพจ</div>
            </div>
            <v-table v-if="salePages.length" density="comfortable" class="rounded-lg">
              <thead>
                <tr>
                  <th class="table-header">เพจ</th>
                  <th class="table-header text-center">Views</th>
                  <th class="table-header text-center">Orders</th>
                  <th class="table-header text-center">Conv.</th>
                  <th class="table-header text-center">Bump</th>
                  <th class="table-header text-end">ยอดขาย</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="sp in salePages" :key="sp.slug">
                  <td class="font-weight-medium">{{ sp.title }}</td>
                  <td class="text-center">{{ sp.views }}</td>
                  <td class="text-center">{{ sp.orders_count }}</td>
                  <td class="text-center">
                    <v-chip size="x-small" variant="tonal" label :color="sp.conv_pct >= 3 ? 'success' : 'default'">
                      {{ sp.conv_pct.toFixed(1) }}%
                    </v-chip>
                  </td>
                  <td class="text-center">{{ sp.bump_count }}</td>
                  <td class="text-end">{{ formatCurrency(sp.revenue) }}</td>
                </tr>
              </tbody>
            </v-table>
            <div v-else class="text-center pa-8" style="color: #6B7280;">
              <v-icon icon="mdi-rocket-launch-outline" size="32" class="mb-2" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มี sale page</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Stock Overview + Recent Orders -->
    <v-row class="mt-1">
      <v-col cols="12" lg="5">
        <v-card>
          <v-card-text class="pa-5">
            <div class="d-flex align-center mb-4">
              <div>
                <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Stock Overview</div>
                <div class="text-caption" style="color: #6B7280;">Current inventory levels</div>
              </div>
              <v-spacer />
              <v-chip v-if="(stats.low_stock_count || 0) > 0" color="error" variant="tonal" size="small" label>
                {{ stats.low_stock_count }} low stock
              </v-chip>
            </div>
            <apexchart
              v-if="stockData.length"
              type="bar"
              height="350"
              :options="stockChartOptions"
              :series="stockChartSeries"
            />
            <div v-else class="text-center pa-12" style="color: #6B7280;">
              <v-icon icon="mdi-package-variant" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">No products yet</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Recent Orders -->
      <v-col cols="12" lg="7">
        <v-card>
          <v-card-text class="pa-5">
            <div class="d-flex align-center mb-4">
              <div>
                <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Recent Orders</div>
                <div class="text-caption" style="color: #6B7280;">Latest transactions</div>
              </div>
              <v-spacer />
              <v-btn variant="tonal" color="secondary" size="small" to="/orders" class="text-none">View All</v-btn>
            </div>

            <v-table density="comfortable" class="rounded-lg">
              <thead>
                <tr>
                  <th class="table-header">Order</th>
                  <th class="table-header">Customer</th>
                  <th class="table-header text-end">Amount</th>
                  <th class="table-header text-center">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="order in stats.recent_orders" :key="order.id">
                  <td class="font-weight-medium">#{{ order.id }}</td>
                  <td>
                    <div class="d-flex align-center">
                      <v-avatar size="28" color="secondary" variant="tonal" class="mr-2">
                        <span class="text-caption font-weight-bold">{{ (order.customer?.name || '?')[0] }}</span>
                      </v-avatar>
                      {{ order.customer?.name || '-' }}
                    </div>
                  </td>
                  <td class="text-end font-weight-medium">{{ formatCurrency(order.total_amount) }}</td>
                  <td class="text-center">
                    <v-chip :color="statusColor(order.status)" size="small" variant="tonal" label>{{ order.status }}</v-chip>
                  </td>
                </tr>
                <tr v-if="!stats.recent_orders?.length">
                  <td colspan="4" class="text-center pa-8" style="color: #6B7280;">
                    <v-icon icon="mdi-receipt-text-outline" size="32" class="mb-2" color="grey" /><br>
                    No orders yet
                  </td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import api from '@/services/api'

const stats = ref<any>({})
const chartData = ref<any>({})
const selectedPeriod = ref('month')
const overview = ref<any>({})
const days = ref('30')

async function fetchCharts() {
  try {
    const { data } = await api.get('/dashboard/charts', { params: { period: selectedPeriod.value } })
    chartData.value = data
  } catch {}
}

async function fetchOverview() {
  try {
    const { data } = await api.get('/analytics/overview', { params: { days: days.value } })
    overview.value = data
  } catch {}
}

watch(selectedPeriod, fetchCharts)
watch(days, fetchOverview)

// ── Stat Cards ──
const statCards = computed(() => [
  {
    title: 'Total Products', value: stats.value.product_count || 0,
    icon: 'mdi-package-variant', subtitle: `${stats.value.low_stock_count || 0} low stock`,
    iconBg: 'rgba(196, 162, 77, 0.12)', iconColor: '#C4A24D',
  },
  {
    title: 'Customers', value: stats.value.customer_count || 0,
    icon: 'mdi-account-group-outline', subtitle: null,
    iconBg: 'rgba(21, 128, 61, 0.10)', iconColor: '#15803D',
  },
  {
    title: 'Total Orders', value: stats.value.order_count || 0,
    icon: 'mdi-receipt-text-outline', subtitle: `${stats.value.pending_order_count || 0} pending`,
    iconBg: 'rgba(42, 120, 214, 0.10)', iconColor: '#2a78d6',
  },
  {
    title: 'Revenue', value: formatCurrency(stats.value.total_revenue || 0),
    icon: 'mdi-trending-up', subtitle: null,
    iconBg: 'rgba(26, 23, 20, 0.08)', iconColor: '#111827',
  },
])

// ── Period KPI cards (current vs previous) ──
function deltaPct(cur: number, prev: number): number | null {
  if (!prev) return null
  return ((cur - prev) / prev) * 100
}

const kpiCards = computed(() => {
  const cur = overview.value.current || {}
  const prev = overview.value.previous || {}
  return [
    {
      title: 'ยอดขาย', value: formatCurrency(cur.revenue || 0),
      delta: deltaPct(cur.revenue || 0, prev.revenue || 0),
      caption: `ช่วงก่อน ${formatCurrency(prev.revenue || 0)}`,
    },
    {
      title: 'ออเดอร์', value: `${cur.orders || 0}`,
      delta: deltaPct(cur.orders || 0, prev.orders || 0),
      caption: `ช่วงก่อน ${prev.orders || 0} ออเดอร์`,
    },
    {
      title: 'ยอดเฉลี่ย/บิล (AOV)', value: formatCurrency(cur.aov || 0),
      delta: deltaPct(cur.aov || 0, prev.aov || 0),
      caption: `${cur.units_sold || 0} ชิ้นที่ขายได้`,
    },
    {
      title: 'กำไรขั้นต้น', value: formatCurrency(cur.gross_profit || 0),
      delta: deltaPct(cur.gross_profit || 0, prev.gross_profit || 0),
      caption: `margin ${(cur.margin_pct || 0).toFixed(1)}%`,
    },
  ]
})

// ── Channel donut ──
// Fixed color per channel (identity), never cycled — palette validated for CVD.
const CHANNEL_META: Record<string, { label: string; color: string }> = {
  facebook: { label: 'Facebook', color: '#2a78d6' },
  line: { label: 'LINE', color: '#008300' },
  instagram: { label: 'Instagram', color: '#4a3aa7' },
  tiktok: { label: 'TikTok', color: '#111827' },
  storefront: { label: 'หน้าเว็บร้าน', color: '#eda100' },
  'sale-page': { label: 'Sale Page', color: '#e34948' },
  'walk-in': { label: 'หน้าร้าน', color: '#6B7280' },
  other: { label: 'อื่นๆ', color: '#9CA3AF' },
  unknown: { label: 'ไม่ระบุ', color: '#C9CED6' },
}
function channelMeta(ch: string) {
  return CHANNEL_META[ch] || { label: ch, color: '#9CA3AF' }
}

const channels = computed(() => overview.value.channels || [])
const channelSeries = computed(() => channels.value.map((c: any) => c.revenue))
const channelOptions = computed(() => ({
  chart: { type: 'donut', fontFamily: 'inherit' },
  labels: channels.value.map((c: any) => channelMeta(c.channel).label),
  colors: channels.value.map((c: any) => channelMeta(c.channel).color),
  stroke: { width: 2, colors: ['#fff'] },
  plotOptions: {
    pie: {
      donut: {
        size: '68%',
        labels: {
          show: true,
          name: { fontSize: '13px', color: '#111827' },
          value: {
            fontSize: '18px', fontWeight: 700, color: '#111827',
            formatter: (val: string) => formatCurrency(Number(val)),
          },
          total: {
            show: true, label: 'รวม', fontSize: '12px', color: '#6B7280',
            formatter: (w: any) => formatCurrency(w.globals.seriesTotals.reduce((a: number, b: number) => a + b, 0)),
          },
        },
      },
    },
  },
  dataLabels: { enabled: false },
  legend: { position: 'bottom', labels: { colors: '#6B7280' }, markers: { offsetX: -4 } },
  tooltip: { theme: 'light', y: { formatter: (val: number) => formatCurrency(val) } },
}))

// ── Category bar ──
const categories = computed(() => overview.value.categories || [])
const categorySeries = computed(() => [{ name: 'ยอดขาย', data: categories.value.map((c: any) => c.revenue) }])
const categoryOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit' },
  plotOptions: { bar: { horizontal: true, barHeight: '60%', borderRadius: 4 } },
  colors: ['#2a78d6'],
  dataLabels: {
    enabled: true,
    style: { fontSize: '11px', fontWeight: 600 },
    formatter: (val: number) => formatCurrency(val),
  },
  xaxis: {
    categories: categories.value.map((c: any) => c.category),
    labels: {
      style: { colors: '#6B7280', fontSize: '11px' },
      formatter: (val: string) => {
        const n = Number(val)
        if (n >= 1000) return `${(n / 1000).toFixed(0)}K`
        return val
      },
    },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: { labels: { style: { colors: '#111827', fontSize: '12px' }, maxWidth: 140 } },
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3, yaxis: { lines: { show: false } } },
  tooltip: {
    theme: 'light',
    y: {
      formatter: (val: number, opts: any) => {
        const units = categories.value[opts.dataPointIndex]?.units || 0
        return `${formatCurrency(val)} (${units} ชิ้น)`
      },
    },
  },
  legend: { show: false },
}))

const coupons = computed(() => overview.value.coupons || [])
const salePages = computed(() => overview.value.sale_pages || [])

// ── Revenue Area Chart ──
const revenueChartSeries = computed(() => [
  {
    name: 'Revenue',
    data: (chartData.value.revenue_series || []).map((p: any) => p.revenue),
  },
  {
    name: 'Orders',
    data: (chartData.value.revenue_series || []).map((p: any) => p.orders),
  },
])

const revenueChartOptions = computed(() => ({
  chart: {
    type: 'area',
    toolbar: { show: false },
    fontFamily: 'inherit',
    zoom: { enabled: false },
  },
  colors: ['#2a78d6', '#1baf7a'],
  fill: {
    type: 'gradient',
    gradient: {
      shadeIntensity: 1,
      opacityFrom: 0.4,
      opacityTo: 0.05,
      stops: [0, 90, 100],
    },
  },
  stroke: { curve: 'smooth', width: [3, 2] },
  dataLabels: { enabled: false },
  xaxis: {
    categories: (chartData.value.revenue_series || []).map((p: any) => p.label),
    labels: {
      style: { colors: '#6B7280', fontSize: '11px' },
      rotate: -45,
      rotateAlways: false,
    },
    axisBorder: { color: '#E5E7EB' },
    axisTicks: { color: '#E5E7EB' },
  },
  yaxis: [
    {
      title: { text: 'Revenue (THB)', style: { color: '#6B7280', fontSize: '11px', fontWeight: 500 } },
      labels: {
        style: { colors: '#6B7280', fontSize: '11px' },
        formatter: (val: number) => {
          if (val >= 1000000) return `${(val / 1000000).toFixed(1)}M`
          if (val >= 1000) return `${(val / 1000).toFixed(0)}K`
          return val.toFixed(0)
        },
      },
    },
    {
      opposite: true,
      title: { text: 'Orders', style: { color: '#6B7280', fontSize: '11px', fontWeight: 500 } },
      labels: { style: { colors: '#6B7280', fontSize: '11px' } },
    },
  ],
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3 },
  tooltip: {
    theme: 'light',
    y: {
      formatter: (val: number, { seriesIndex }: any) => {
        if (seriesIndex === 0) return formatCurrency(val)
        return `${val} orders`
      },
    },
  },
  legend: {
    position: 'top',
    horizontalAlign: 'right',
    labels: { colors: '#6B7280' },
    markers: { offsetX: -4 },
  },
}))

// ── Order Status Donut ──
const orderStatusSeries = computed(() => {
  const data = chartData.value.order_status || []
  return data.map((s: any) => s.count)
})

const orderStatusOptions = computed(() => ({
  chart: { type: 'donut', fontFamily: 'inherit' },
  labels: ['Pending', 'Confirmed', 'Shipped', 'Delivered', 'Cancelled'],
  colors: ['#eda100', '#2a78d6', '#4a3aa7', '#008300', '#e34948'],
  stroke: { width: 2, colors: ['#fff'] },
  plotOptions: {
    pie: {
      donut: {
        size: '68%',
        labels: {
          show: true,
          name: { fontSize: '13px', color: '#111827' },
          value: { fontSize: '22px', fontWeight: 700, color: '#111827' },
          total: {
            show: true,
            label: 'Total',
            fontSize: '12px',
            color: '#6B7280',
            formatter: (w: any) => w.globals.seriesTotals.reduce((a: number, b: number) => a + b, 0),
          },
        },
      },
    },
  },
  dataLabels: { enabled: false },
  legend: {
    position: 'bottom',
    labels: { colors: '#6B7280' },
    markers: { offsetX: -4 },
  },
  tooltip: { theme: 'light' },
}))

// ── Stock Overview Bar Chart ──
const stockData = computed(() => chartData.value.stock_overview || [])

const stockChartSeries = computed(() => [{
  name: 'Stock',
  data: stockData.value.map((p: any) => p.stock),
}])

const stockChartOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit' },
  plotOptions: {
    bar: {
      horizontal: true,
      barHeight: '65%',
      borderRadius: 4,
      distributed: true,
    },
  },
  colors: stockData.value.map((p: any) => {
    if (p.stock === 0) return '#e34948'
    if (p.stock <= 5) return '#eda100'
    return '#1baf7a'
  }),
  dataLabels: {
    enabled: true,
    style: { fontSize: '11px', fontWeight: 600 },
    formatter: (val: number) => `${val} pcs`,
  },
  xaxis: {
    labels: { style: { colors: '#6B7280', fontSize: '11px' } },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: {
    labels: {
      style: { colors: '#111827', fontSize: '12px' },
      maxWidth: 150,
    },
  },
  categories: stockData.value.map((p: any) => p.name),
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3, xaxis: { lines: { show: true } }, yaxis: { lines: { show: false } } },
  tooltip: {
    theme: 'light',
    y: { formatter: (val: number) => `${val} pieces` },
  },
  legend: { show: false },
}))

// ── Top Selling Products ──
const topSellingData = computed(() => chartData.value.top_selling_products || [])

const topSellingSeries = computed(() => [
  { name: 'Qty Sold', data: topSellingData.value.map((p: any) => p.quantity) },
  { name: 'Revenue', data: topSellingData.value.map((p: any) => p.revenue) },
])

const topSellingOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit' },
  plotOptions: {
    bar: { horizontal: false, columnWidth: '55%', borderRadius: 5 },
  },
  colors: ['#2a78d6', '#1baf7a'],
  dataLabels: { enabled: false },
  xaxis: {
    categories: topSellingData.value.map((p: any) => p.name),
    labels: {
      style: { colors: '#6B7280', fontSize: '11px' },
      rotate: -30,
      trim: true,
      maxHeight: 60,
    },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: [
    {
      title: { text: 'Quantity', style: { color: '#6B7280', fontSize: '11px', fontWeight: 500 } },
      labels: { style: { colors: '#6B7280', fontSize: '11px' } },
    },
    {
      opposite: true,
      title: { text: 'Revenue (THB)', style: { color: '#6B7280', fontSize: '11px', fontWeight: 500 } },
      labels: {
        style: { colors: '#6B7280', fontSize: '11px' },
        formatter: (val: number) => {
          if (val >= 1000) return `${(val / 1000).toFixed(0)}K`
          return val.toFixed(0)
        },
      },
    },
  ],
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3 },
  tooltip: {
    theme: 'light',
    y: {
      formatter: (val: number, { seriesIndex }: any) => {
        if (seriesIndex === 1) return formatCurrency(val)
        return `${val} pcs`
      },
    },
  },
  legend: {
    position: 'top',
    horizontalAlign: 'right',
    labels: { colors: '#6B7280' },
    markers: { offsetX: -4 },
  },
}))

// ── Helpers ──
function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n)
}

function statusColor(status: string) {
  const map: Record<string, string> = {
    pending: 'warning', confirmed: 'info', shipped: 'secondary', delivered: 'success', cancelled: 'error'
  }
  return map[status] || 'grey'
}

onMounted(async () => {
  const [statsRes] = await Promise.all([
    api.get('/dashboard'),
    fetchCharts(),
    fetchOverview(),
  ])
  stats.value = statsRes.data
})
</script>

<style scoped>
.stat-card {
  transition: transform 0.2s, box-shadow 0.2s;
  border: 1px solid #E5E7EB !important;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(26, 23, 20, 0.06) !important;
}

.stat-label {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: #6B7280;
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.kpi-tile {
  border: 1px solid #E5E7EB;
  border-radius: 12px;
  height: 100%;
}

.table-header {
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: #6B7280 !important;
}

.period-toggle {
  border-radius: 8px !important;
  overflow: hidden;
}
</style>
