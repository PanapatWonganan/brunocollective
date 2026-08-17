<template>
  <div>
    <div class="d-flex align-center mb-4">
      <div>
        <div class="text-h5 font-weight-bold">Broadcast</div>
        <div class="text-caption text-medium-emphasis">
          ส่งข้อความหาลูกค้าทาง LINE ตามกลุ่ม (RFM segment เดียวกับหน้า Analytics)
        </div>
      </div>
      <v-spacer />
      <v-btn icon="mdi-refresh" variant="text" :loading="loading" @click="fetchAll" />
    </div>

    <v-alert v-if="audienceLoaded && !lineEnabled" type="warning" variant="tonal" class="mb-4">
      ยังไม่ได้ตั้งค่า LINE (LINE_CHANNEL_ACCESS_TOKEN) — ตั้งค่าก่อนจึงจะส่ง broadcast ได้
    </v-alert>

    <v-row>
      <!-- Audience -->
      <v-col cols="12" md="5">
        <v-card class="pa-4">
          <div class="text-subtitle-1 font-weight-bold mb-1">กลุ่มเป้าหมาย</div>
          <div class="text-caption text-medium-emphasis mb-3">
            นับเฉพาะลูกค้าที่ทักแชท LINE เข้ามา (ส่งหาได้จริง) — เรียงตามยอดซื้อในแต่ละกลุ่ม
          </div>
          <template v-if="segments.length">
            <v-checkbox
              v-for="seg in segments" :key="seg.segment"
              v-model="selectedSegments" :value="seg.segment"
              density="compact" hide-details class="mb-1"
            >
              <template v-slot:label>
                <div>
                  <span class="font-weight-medium">{{ seg.segment }}</span>
                  <v-chip size="x-small" variant="tonal" color="primary" class="ml-2">{{ seg.count }} คน</v-chip>
                  <div class="text-caption text-medium-emphasis">{{ segmentHint(seg.segment) }}</div>
                </div>
              </template>
            </v-checkbox>
          </template>
          <div v-else-if="audienceLoaded" class="text-center pa-6 text-medium-emphasis text-body-2">
            ยังไม่มีลูกค้าที่ทัก LINE เข้ามา
          </div>
        </v-card>
      </v-col>

      <!-- Composer -->
      <v-col cols="12" md="7">
        <v-card class="pa-4">
          <div class="text-subtitle-1 font-weight-bold mb-3">ข้อความ</div>
          <v-textarea
            v-model="message" rows="6" auto-grow
            placeholder="เช่น สวัสดีค่ะคุณ {name} 💛 คอลเลคชันใหม่มาแล้ว ใช้โค้ด COMEBACK ลด 10% ถึงศุกร์นี้..."
            hint="ใช้ {name} แทนชื่อลูกค้าได้ • ค่าส่งข้อความคิดตามแพ็กเกจ LINE OA ของร้าน"
            persistent-hint
          />
          <div class="d-flex align-center mt-4">
            <div class="text-body-2">
              จะส่งถึง <strong>{{ recipientCount }}</strong> คน
              <span v-if="selectedSegments.length" class="text-medium-emphasis">
                ({{ selectedSegments.join(', ') }})
              </span>
            </div>
            <v-spacer />
            <v-btn
              color="primary" class="text-none px-6" prepend-icon="mdi-send"
              :disabled="!canSend" :loading="sending"
              @click="confirmDialog = true"
            >ส่ง Broadcast</v-btn>
          </div>
          <v-alert v-if="sendError" type="error" variant="tonal" density="compact" class="mt-3">
            {{ sendError }}
          </v-alert>
        </v-card>
      </v-col>
    </v-row>

    <!-- History -->
    <v-card class="mt-4">
      <v-card-title class="text-subtitle-1 font-weight-bold">ประวัติการส่ง</v-card-title>
      <v-data-table
        :headers="historyHeaders" :items="history" :loading="loading"
        density="comfortable" items-per-page="10"
      >
        <template v-slot:item.created_at="{ item }">
          <span class="text-caption">{{ formatDate(item.created_at) }}</span>
        </template>
        <template v-slot:item.segments="{ item }">
          <v-chip v-for="s in item.segments || []" :key="s" size="x-small" label variant="tonal" color="secondary" class="mr-1">
            {{ s }}
          </v-chip>
        </template>
        <template v-slot:item.message="{ item }">
          <span class="text-caption">{{ item.message.length > 60 ? item.message.slice(0, 60) + '…' : item.message }}</span>
        </template>
        <template v-slot:item.progress="{ item }">
          <span class="text-caption">
            {{ item.sent }}/{{ item.total }}
            <span v-if="item.failed" class="text-error">(พลาด {{ item.failed }})</span>
          </span>
        </template>
        <template v-slot:item.status="{ item }">
          <v-chip size="small" label variant="tonal" :color="item.status === 'done' ? 'success' : 'info'">
            {{ item.status === 'done' ? 'ส่งแล้ว' : 'กำลังส่ง…' }}
          </v-chip>
        </template>
        <template v-slot:no-data>
          <div class="text-center pa-6 text-medium-emphasis text-body-2">ยังไม่เคยส่ง broadcast</div>
        </template>
      </v-data-table>
    </v-card>

    <!-- Confirm dialog -->
    <v-dialog v-model="confirmDialog" max-width="440">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">ยืนยันการส่ง</span>
        </v-card-title>
        <v-card-text class="px-5">
          <div class="text-body-2 mb-3">
            ส่งข้อความถึง <strong>{{ recipientCount }} คน</strong> ในกลุ่ม
            <strong>{{ selectedSegments.join(', ') }}</strong> — ส่งแล้วยกเลิกไม่ได้
          </div>
          <v-card variant="tonal" class="pa-3 text-body-2" style="white-space: pre-wrap;">{{ message }}</v-card>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="confirmDialog = false">ยกเลิก</v-btn>
          <v-btn color="primary" class="text-none px-6" :loading="sending" @click="send">ส่งเลย</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar" timeout="5000" color="success">{{ snackbarText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import api from '@/services/api'

interface Recipient { conversation_id: number; customer_id: number; name: string; segment: string; orders: number; spent: number }
interface Segment { segment: string; count: number; recipients: Recipient[] }
interface Broadcast {
  id: number; message: string; segments: string[] | null;
  total: number; sent: number; failed: number; status: string; created_at: string;
}

const loading = ref(false)
const audienceLoaded = ref(false)
const lineEnabled = ref(true)
const segments = ref<Segment[]>([])
const selectedSegments = ref<string[]>([])
const message = ref('')
const sending = ref(false)
const sendError = ref('')
const confirmDialog = ref(false)
const history = ref<Broadcast[]>([])
const snackbar = ref(false)
const snackbarText = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const historyHeaders = [
  { title: 'วันที่', key: 'created_at' },
  { title: 'กลุ่ม', key: 'segments', sortable: false },
  { title: 'ข้อความ', key: 'message', sortable: false },
  { title: 'ส่งแล้ว', key: 'progress', sortable: false },
  { title: 'สถานะ', key: 'status' },
]

const recipientCount = computed(() =>
  segments.value
    .filter(s => selectedSegments.value.includes(s.segment))
    .reduce((sum, s) => sum + s.count, 0))

const canSend = computed(() =>
  lineEnabled.value && message.value.trim().length > 0 && recipientCount.value > 0 && !sending.value)

function segmentHint(seg: string): string {
  return ({
    'VIP': 'ซื้อ 3 ครั้งขึ้นไป และซื้อล่าสุดใน 60 วัน',
    'ขาประจำ': 'ซื้อ 2 ครั้งขึ้นไปใน 90 วัน',
    'ลูกค้าใหม่': 'เพิ่งซื้อครั้งแรก',
    'ทั่วไป': 'ลูกค้าทั่วไป',
    'กำลังจะหาย': 'ไม่ได้ซื้อมา 3-6 เดือน — เหมาะกับโค้ด win-back',
    'หายไปแล้ว': 'ไม่ได้ซื้อเกิน 6 เดือน',
    'ยังไม่เคยซื้อ': 'ผูกลูกค้าแล้วแต่ยังไม่มีออเดอร์',
    'ยังไม่ผูกลูกค้า': 'ทัก LINE เข้ามาแต่ยังไม่ได้ผูกกับลูกค้าในระบบ',
  } as Record<string, string>)[seg] || ''
}

async function fetchAudience() {
  const { data } = await api.get('/broadcasts/audience')
  lineEnabled.value = !!data.line_enabled
  segments.value = data.segments || []
  audienceLoaded.value = true
}

async function fetchHistory() {
  const { data } = await api.get('/broadcasts')
  history.value = data || []
}

async function fetchAll() {
  loading.value = true
  try {
    await Promise.all([fetchAudience(), fetchHistory()])
  } finally {
    loading.value = false
  }
}

async function send() {
  sending.value = true
  sendError.value = ''
  try {
    await api.post('/broadcasts', {
      segments: selectedSegments.value,
      message: message.value.trim(),
    })
    confirmDialog.value = false
    snackbarText.value = `เริ่มส่งถึง ${recipientCount.value} คนแล้ว — ดูความคืบหน้าในประวัติด้านล่าง`
    snackbar.value = true
    message.value = ''
    selectedSegments.value = []
    fetchHistory()
  } catch (err: any) {
    sendError.value = err.response?.data?.error || 'ส่งไม่สำเร็จ'
    confirmDialog.value = false
  } finally {
    sending.value = false
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString('th-TH', {
    day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit',
  })
}

onMounted(() => {
  fetchAll()
  // Poll while a broadcast is sending so sent/failed tick up live.
  pollTimer = setInterval(() => {
    if (history.value.some(b => b.status === 'sending')) fetchHistory()
  }, 4000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
