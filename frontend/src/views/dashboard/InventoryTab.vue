<template>
  <div>
    <!-- Stock health tiles -->
    <v-row>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">มูลค่าสต็อกทั้งหมด</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">{{ formatCurrency(data.total_stock_value || 0) }}</div>
            <div class="text-caption mt-1" style="color: #6B7280;">คิดจากต้นทุน (ถ้าไม่มีต้นทุนใช้ราคาขาย)</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">เงินจมใน Dead Stock</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #DC2626;">{{ formatCurrency(data.dead_stock_value || 0) }}</div>
            <div class="text-caption mt-1" style="color: #6B7280;">สินค้าไม่ขยับเกิน 60 วัน</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">สินค้าไม่ขยับ (>60 วัน)</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #111827;">{{ deadStockProducts.length }} รายการ</div>
            <div class="text-caption mt-1" style="color: #6B7280;">ควรจัดโปรระบาย / ทำ sale page</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" lg="3">
        <v-card class="stat-card" border="false">
          <v-card-text class="pa-5">
            <div class="stat-label">ใกล้หมด (ขายได้อีก <14 วัน)</div>
            <div class="text-h5 font-weight-bold mt-2" style="color: #B45309;">{{ lowCoverProducts.length }} รายการ</div>
            <div class="text-caption mt-1" style="color: #6B7280;">จาก run rate 30 วันล่าสุด — ควรสั่งเติม</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Size curve + Colors -->
    <v-row class="mt-1">
      <v-col cols="12" md="7">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="d-flex align-center mb-2 flex-wrap ga-2">
              <div>
                <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">Size Curve — ขายแล้ว vs คงเหลือ</div>
                <div class="text-caption" style="color: #6B7280;">ไซส์ไหนขายดี ไซส์ไหนค้างสต็อก — ใช้วางแผนสั่งผลิตรอบหน้า</div>
              </div>
              <v-spacer />
              <v-select
                v-model="categoryFilter" :items="categoryChoices" label="หมวด" density="compact"
                hide-details clearable style="max-width: 180px;"
              />
            </div>
            <apexchart v-if="sizeCurve.length" type="bar" height="300" :options="sizeCurveOptions" :series="sizeCurveSeries" />
            <div v-else class="text-center pa-12" style="color: #6B7280;">
              <v-icon icon="mdi-ruler" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มีข้อมูลไซส์</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="5">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-2">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">สีขายดี</div>
              <div class="text-caption" style="color: #6B7280;">จำนวนชิ้นที่ขายได้ต่อสี</div>
            </div>
            <apexchart v-if="colors.length" type="bar" height="300" :options="colorOptions" :series="colorSeries" />
            <div v-else class="text-center pa-12" style="color: #6B7280;">
              <v-icon icon="mdi-palette-outline" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่มีข้อมูลสี</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Product movement table -->
    <v-card class="mt-4">
      <v-card-text class="pa-5">
        <div class="d-flex align-center mb-4 flex-wrap ga-2">
          <div>
            <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">การเคลื่อนไหวรายสินค้า</div>
            <div class="text-caption" style="color: #6B7280;">เรียงจากสินค้าที่ไม่ขยับนานที่สุด</div>
          </div>
          <v-spacer />
          <v-btn-toggle v-model="tableFilter" density="compact" color="secondary" variant="outlined" class="period-toggle">
            <v-btn value="all" size="small" class="text-none">ทั้งหมด</v-btn>
            <v-btn value="dead" size="small" class="text-none">Dead Stock</v-btn>
            <v-btn value="reorder" size="small" class="text-none">ควรสั่งเติม</v-btn>
          </v-btn-toggle>
        </div>

        <v-data-table
          :headers="tableHeaders"
          :items="filteredProducts"
          items-per-page="10"
          class="rounded-lg"
        >
          <template v-slot:item.name="{ item }">
            <div class="font-weight-medium">{{ item.name }}</div>
            <div class="text-caption text-medium-emphasis">{{ [item.sku, item.category].filter(Boolean).join(' · ') }}</div>
          </template>
          <template v-slot:item.stock="{ item }">
            <v-chip :color="item.stock === 0 ? 'error' : item.stock <= 5 ? 'warning' : 'success'" variant="tonal" size="small" label>
              {{ item.stock }}
            </v-chip>
          </template>
          <template v-slot:item.last_sold_days="{ item }">
            <span v-if="item.last_sold_days === null" class="text-medium-emphasis">ไม่เคยขาย</span>
            <span v-else>{{ item.last_sold_days }} วันก่อน</span>
          </template>
          <template v-slot:item.sell_through="{ item }">
            <div class="d-flex align-center ga-2">
              <v-progress-linear
                :model-value="item.sell_through" height="6" rounded style="min-width: 60px; max-width: 80px;"
                :color="item.sell_through >= 70 ? 'success' : item.sell_through >= 40 ? 'secondary' : 'error'"
              />
              <span class="text-caption">{{ item.sell_through.toFixed(0) }}%</span>
            </div>
          </template>
          <template v-slot:item.cover_days="{ item }">
            <span v-if="item.cover_days === null" class="text-medium-emphasis">—</span>
            <v-chip v-else size="small" variant="tonal" label :color="item.cover_days < 14 ? 'warning' : 'default'">
              ~{{ item.cover_days }} วัน
            </v-chip>
          </template>
          <template v-slot:item.stock_value="{ item }">
            {{ formatCurrency(item.stock_value) }}
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import api from '@/services/api'

interface InvProduct {
  id: number; name: string; sku: string; category: string;
  stock: number; price: number; cost: number; stock_value: number;
  sold_30: number; sold_total: number; last_sold_days: number | null;
  age_days: number; sell_through: number; cover_days: number | null;
}

const data = ref<any>({})
const categoryFilter = ref<string | null>(null)
const tableFilter = ref('all')

async function fetchInventory() {
  try {
    const params: any = {}
    if (categoryFilter.value) params.category = categoryFilter.value
    const { data: res } = await api.get('/analytics/inventory', { params })
    data.value = res
  } catch {}
}

watch(categoryFilter, fetchInventory)
onMounted(fetchInventory)

const products = computed<InvProduct[]>(() => data.value.products || [])
const sizeCurve = computed(() => data.value.size_curve || [])
const colors = computed(() => data.value.colors || [])

const categoryChoices = computed(() => {
  const set = new Set<string>()
  for (const p of products.value) if (p.category) set.add(p.category)
  return Array.from(set)
})

const deadStockProducts = computed(() =>
  products.value.filter(p => p.stock > 0 &&
    ((p.last_sold_days !== null && p.last_sold_days > 60) || (p.last_sold_days === null && p.age_days > 60)))
)

const lowCoverProducts = computed(() =>
  products.value.filter(p => p.cover_days !== null && p.cover_days < 14)
)

const filteredProducts = computed<InvProduct[]>(() => {
  if (tableFilter.value === 'dead') return deadStockProducts.value
  if (tableFilter.value === 'reorder') return lowCoverProducts.value
  return products.value
})

const tableHeaders = [
  { title: 'สินค้า', key: 'name' },
  { title: 'คงเหลือ', key: 'stock', align: 'center' as const },
  { title: 'ขายได้ 30 วัน', key: 'sold_30', align: 'center' as const },
  { title: 'ขายล่าสุด', key: 'last_sold_days', align: 'center' as const },
  { title: 'Sell-through', key: 'sell_through', align: 'center' as const },
  { title: 'สต็อกพออีก', key: 'cover_days', align: 'center' as const },
  { title: 'มูลค่าคงเหลือ', key: 'stock_value', align: 'end' as const },
]

// ── Size curve: grouped bars, sold (gold) vs remaining stock (blue) ──
const sizeCurveSeries = computed(() => [
  { name: 'ขายแล้ว', data: sizeCurve.value.map((s: any) => s.sold) },
  { name: 'คงเหลือ', data: sizeCurve.value.map((s: any) => s.stock) },
])

const sizeCurveOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit' },
  plotOptions: { bar: { horizontal: false, columnWidth: '60%', borderRadius: 4 } },
  colors: ['#2a78d6', '#1baf7a'],
  dataLabels: { enabled: false },
  xaxis: {
    categories: sizeCurve.value.map((s: any) => s.size),
    labels: { style: { colors: '#6B7280', fontSize: '12px' } },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: { labels: { style: { colors: '#6B7280', fontSize: '11px' } } },
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3 },
  tooltip: { theme: 'light', y: { formatter: (val: number) => `${val} ชิ้น` } },
  legend: { position: 'top', horizontalAlign: 'right', labels: { colors: '#6B7280' }, markers: { offsetX: -4 } },
}))

// ── Colors sold: horizontal single-hue bar ──
const colorSeries = computed(() => [{ name: 'ขายแล้ว', data: colors.value.map((c: any) => c.sold) }])

const colorOptions = computed(() => ({
  chart: { type: 'bar', toolbar: { show: false }, fontFamily: 'inherit' },
  plotOptions: { bar: { horizontal: true, barHeight: '60%', borderRadius: 4 } },
  colors: ['#2a78d6'],
  dataLabels: {
    enabled: true,
    style: { fontSize: '11px', fontWeight: 600 },
    formatter: (val: number) => `${val} ชิ้น`,
  },
  xaxis: {
    categories: colors.value.map((c: any) => c.color),
    labels: { style: { colors: '#6B7280', fontSize: '11px' } },
    axisBorder: { color: '#E5E7EB' },
  },
  yaxis: { labels: { style: { colors: '#111827', fontSize: '12px' }, maxWidth: 120 } },
  grid: { borderColor: '#EEF1F4', strokeDashArray: 3, yaxis: { lines: { show: false } } },
  tooltip: {
    theme: 'light',
    y: {
      formatter: (val: number, opts: any) => {
        const stock = colors.value[opts.dataPointIndex]?.stock || 0
        return `ขายแล้ว ${val} ชิ้น · คงเหลือ ${stock} ชิ้น`
      },
    },
  },
  legend: { show: false },
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
.period-toggle {
  border-radius: 8px !important;
  overflow: hidden;
}
</style>
