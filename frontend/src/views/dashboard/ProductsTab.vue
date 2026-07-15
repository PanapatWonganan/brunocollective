<template>
  <div>
    <v-row>
      <!-- ABC classification -->
      <v-col cols="12" lg="7">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-3">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ABC Analysis — สินค้าตัวไหนเลี้ยงร้าน</div>
              <div class="text-caption" style="color: #6B7280;">
                A = ทำรายได้ 80% แรก (ห้ามให้ขาดสต็อก) · B = 15% ถัดมา · C = ที่เหลือ (พิจารณาเลิกทำ)
              </div>
            </div>

            <div class="d-flex ga-4 mb-4">
              <div v-for="cls in abcSummary" :key="cls.label" class="abc-tile pa-3 flex-grow-1" :style="{ borderColor: cls.color + '66' }">
                <div class="d-flex align-center ga-2">
                  <v-avatar size="28" :color="cls.color" variant="tonal">
                    <span class="text-caption font-weight-bold">{{ cls.label }}</span>
                  </v-avatar>
                  <div>
                    <div class="font-weight-bold" style="color: #111827;">{{ cls.count }} สินค้า</div>
                    <div class="text-caption" style="color: #6B7280;">{{ formatCurrency(cls.revenue) }}</div>
                  </div>
                </div>
              </div>
            </div>

            <v-data-table
              :headers="abcHeaders"
              :items="abc"
              items-per-page="10"
              class="rounded-lg"
            >
              <template v-slot:item.name="{ item }">
                <div class="font-weight-medium">{{ item.name }}</div>
                <div v-if="item.category" class="text-caption text-medium-emphasis">{{ item.category }}</div>
              </template>
              <template v-slot:item.class="{ item }">
                <v-chip size="small" variant="tonal" label :color="abcColor(item.class)">{{ item.class }}</v-chip>
              </template>
              <template v-slot:item.revenue="{ item }">
                {{ formatCurrency(item.revenue) }}
              </template>
              <template v-slot:item.share_pct="{ item }">
                {{ item.share_pct.toFixed(1) }}%
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Bought together -->
      <v-col cols="12" lg="5">
        <v-card class="fill-height">
          <v-card-text class="pa-5">
            <div class="mb-3">
              <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">สินค้าที่มักถูกซื้อคู่กัน</div>
              <div class="text-caption" style="color: #6B7280;">
                ใช้จัดเซ็ต ตั้ง order bump ใน sale page หรือแนะนำตอนแชท
              </div>
            </div>
            <v-table v-if="pairs.length" density="comfortable" class="rounded-lg">
              <thead>
                <tr>
                  <th class="table-header">คู่สินค้า</th>
                  <th class="table-header text-center">ซื้อคู่กัน</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(p, i) in pairs" :key="i">
                  <td>
                    <div class="d-flex align-center flex-wrap ga-1 py-2">
                      <v-chip size="small" variant="tonal" color="secondary" label>{{ p.product_a }}</v-chip>
                      <v-icon icon="mdi-plus" size="14" color="grey" />
                      <v-chip size="small" variant="tonal" color="secondary" label>{{ p.product_b }}</v-chip>
                    </div>
                  </td>
                  <td class="text-center font-weight-medium">{{ p.count }} ครั้ง</td>
                </tr>
              </tbody>
            </v-table>
            <div v-else class="text-center pa-8" style="color: #6B7280;">
              <v-icon icon="mdi-link-variant" size="32" class="mb-2" color="grey-lighten-1" />
              <div class="text-body-2">ยังไม่พบคู่สินค้าที่ซื้อซ้ำกันบ่อย (ต้องมีอย่างน้อย 2 ออเดอร์ที่ซื้อคู่เดียวกัน)</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/services/api'

interface AbcItem {
  id: number; name: string; category: string;
  revenue: number; units: number; share_pct: number; cum_pct: number; class: string;
}

const data = ref<any>({})

async function fetchProducts() {
  try {
    const { data: res } = await api.get('/analytics/products')
    data.value = res
  } catch {}
}
onMounted(fetchProducts)

const abc = computed<AbcItem[]>(() => data.value.abc || [])
const pairs = computed(() => data.value.bought_together || [])

const ABC_COLORS: Record<string, string> = { A: '#15803D', B: '#B45309', C: '#DC2626' }
function abcColor(cls: string) {
  const map: Record<string, string> = { A: 'success', B: 'warning', C: 'error' }
  return map[cls] || 'grey'
}

const abcSummary = computed(() =>
  (['A', 'B', 'C'] as const).map(label => {
    const items = abc.value.filter((p: any) => p.class === label)
    return {
      label,
      color: ABC_COLORS[label],
      count: items.length,
      revenue: items.reduce((sum: number, p: any) => sum + p.revenue, 0),
    }
  })
)

const abcHeaders = [
  { title: 'สินค้า', key: 'name' },
  { title: 'Class', key: 'class', align: 'center' as const },
  { title: 'ขายได้', key: 'units', align: 'center' as const },
  { title: 'รายได้', key: 'revenue', align: 'end' as const },
  { title: 'สัดส่วน', key: 'share_pct', align: 'end' as const },
]

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n)
}
</script>

<style scoped>
.table-header {
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: #6B7280 !important;
}
.abc-tile {
  border: 1px solid #E5E7EB;
  border-radius: 12px;
}
</style>
