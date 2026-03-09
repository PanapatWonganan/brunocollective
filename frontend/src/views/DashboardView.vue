<template>
  <div>
    <!-- Stat Cards -->
    <v-row>
      <v-col v-for="card in statCards" :key="card.title" cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="d-flex justify-space-between align-start">
              <div>
                <div class="stat-label">{{ card.title }}</div>
                <div class="text-h4 font-weight-bold mt-2" style="color: #1A1714;">{{ card.value }}</div>
              </div>
              <div class="stat-icon" :style="{ background: card.iconBg }">
                <v-icon :icon="card.icon" size="22" :color="card.iconColor" />
              </div>
            </div>
            <div v-if="card.subtitle" class="text-caption mt-3" style="color: #8C8478;">{{ card.subtitle }}</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Revenue Chart -->
    <v-row class="mt-1">
      <v-col cols="12">
        <v-card>
          <v-card-text class="pa-5">
            <div class="d-flex align-center mb-4 flex-wrap ga-2">
              <div>
                <div class="text-subtitle-1 font-weight-bold" style="color: #1A1714;">Revenue Overview</div>
                <div class="text-caption" style="color: #8C8478;">Sales performance over time</div>
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

    <!-- Charts Row: Order Status + Top Products -->
    <v-row class="mt-1">
      <v-col cols="12" md="5">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #1A1714;">Order Status</div>
              <div class="text-caption" style="color: #8C8478;">Distribution by status</div>
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
              <div class="text-subtitle-1 font-weight-bold" style="color: #1A1714;">Top Selling Products</div>
              <div class="text-caption" style="color: #8C8478;">By quantity sold</div>
            </div>
            <apexchart
              v-if="topSellingData.length"
              type="bar"
              height="280"
              :options="topSellingOptions"
              :series="topSellingSeries"
            />
            <div v-else class="text-center pa-12" style="color: #8C8478;">
              <v-icon icon="mdi-chart-bar" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">No sales data yet</div>
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
                <div class="text-subtitle-1 font-weight-bold" style="color: #1A1714;">Stock Overview</div>
                <div class="text-caption" style="color: #8C8478;">Current inventory levels</div>
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
            <div v-else class="text-center pa-12" style="color: #8C8478;">
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
                <div class="text-subtitle-1 font-weight-bold" style="color: #1A1714;">Recent Orders</div>
                <div class="text-caption" style="color: #8C8478;">Latest transactions</div>
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
                  <td colspan="4" class="text-center pa-8" style="color: #8C8478;">
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

// Fetch chart data when period changes
async function fetchCharts() {
  try {
    const { data } = await api.get('/dashboard/charts', { params: { period: selectedPeriod.value } })
    chartData.value = data
  } catch {}
}

watch(selectedPeriod, fetchCharts)

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
    iconBg: 'rgba(93, 122, 95, 0.12)', iconColor: '#5D7A5F',
  },
  {
    title: 'Total Orders', value: stats.value.order_count || 0,
    icon: 'mdi-receipt-text-outline', subtitle: `${stats.value.pending_order_count || 0} pending`,
    iconBg: 'rgba(107, 123, 141, 0.12)', iconColor: '#6B7B8D',
  },
  {
    title: 'Revenue', value: formatCurrency(stats.value.total_revenue || 0),
    icon: 'mdi-trending-up', subtitle: null,
    iconBg: 'rgba(26, 23, 20, 0.08)', iconColor: '#1A1714',
  },
])

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
  colors: ['#C4A24D', '#6B7B8D'],
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
      style: { colors: '#8C8478', fontSize: '11px' },
      rotate: -45,
      rotateAlways: false,
    },
    axisBorder: { color: '#E8E2D9' },
    axisTicks: { color: '#E8E2D9' },
  },
  yaxis: [
    {
      title: { text: 'Revenue (THB)', style: { color: '#8C8478', fontSize: '11px', fontWeight: 500 } },
      labels: {
        style: { colors: '#8C8478', fontSize: '11px' },
        formatter: (val: number) => {
          if (val >= 1000000) return `${(val / 1000000).toFixed(1)}M`
          if (val >= 1000) return `${(val / 1000).toFixed(0)}K`
          return val.toFixed(0)
        },
      },
    },
    {
      opposite: true,
      title: { text: 'Orders', style: { color: '#8C8478', fontSize: '11px', fontWeight: 500 } },
      labels: { style: { colors: '#8C8478', fontSize: '11px' } },
    },
  ],
  grid: { borderColor: '#F0EBE4', strokeDashArray: 3 },
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
    labels: { colors: '#8C8478' },
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
  colors: ['#D4A843', '#6B7B8D', '#C4A24D', '#5D7A5F', '#9B3B3B'],
  stroke: { width: 2, colors: ['#fff'] },
  plotOptions: {
    pie: {
      donut: {
        size: '68%',
        labels: {
          show: true,
          name: { fontSize: '13px', color: '#1A1714' },
          value: { fontSize: '22px', fontWeight: 700, color: '#1A1714' },
          total: {
            show: true,
            label: 'Total',
            fontSize: '12px',
            color: '#8C8478',
            formatter: (w: any) => w.globals.seriesTotals.reduce((a: number, b: number) => a + b, 0),
          },
        },
      },
    },
  },
  dataLabels: { enabled: false },
  legend: {
    position: 'bottom',
    labels: { colors: '#8C8478' },
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
    if (p.stock === 0) return '#9B3B3B'
    if (p.stock <= 5) return '#D4A843'
    if (p.stock <= 15) return '#C4A24D'
    return '#5D7A5F'
  }),
  dataLabels: {
    enabled: true,
    style: { fontSize: '11px', fontWeight: 600 },
    formatter: (val: number) => `${val} pcs`,
  },
  xaxis: {
    labels: { style: { colors: '#8C8478', fontSize: '11px' } },
    axisBorder: { color: '#E8E2D9' },
  },
  yaxis: {
    labels: {
      style: { colors: '#1A1714', fontSize: '12px' },
      maxWidth: 150,
    },
  },
  categories: stockData.value.map((p: any) => p.name),
  grid: { borderColor: '#F0EBE4', strokeDashArray: 3, xaxis: { lines: { show: true } }, yaxis: { lines: { show: false } } },
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
  colors: ['#C4A24D', '#1A1714'],
  dataLabels: { enabled: false },
  xaxis: {
    categories: topSellingData.value.map((p: any) => p.name),
    labels: {
      style: { colors: '#8C8478', fontSize: '11px' },
      rotate: -30,
      trim: true,
      maxHeight: 60,
    },
    axisBorder: { color: '#E8E2D9' },
  },
  yaxis: [
    {
      title: { text: 'Quantity', style: { color: '#8C8478', fontSize: '11px', fontWeight: 500 } },
      labels: { style: { colors: '#8C8478', fontSize: '11px' } },
    },
    {
      opposite: true,
      title: { text: 'Revenue (THB)', style: { color: '#8C8478', fontSize: '11px', fontWeight: 500 } },
      labels: {
        style: { colors: '#8C8478', fontSize: '11px' },
        formatter: (val: number) => {
          if (val >= 1000) return `${(val / 1000).toFixed(0)}K`
          return val.toFixed(0)
        },
      },
    },
  ],
  grid: { borderColor: '#F0EBE4', strokeDashArray: 3 },
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
    labels: { colors: '#8C8478' },
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
  ])
  stats.value = statsRes.data
})
</script>

<style scoped>
.stat-card {
  transition: transform 0.2s, box-shadow 0.2s;
  border: 1px solid #E8E2D9 !important;
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
  color: #8C8478;
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

.table-header {
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: #8C8478 !important;
}

.period-toggle {
  border-radius: 8px !important;
  overflow: hidden;
}
</style>
