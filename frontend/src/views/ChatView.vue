<template>
  <div>
    <div class="d-flex align-center mb-4">
      <div>
        <div class="text-h5 font-weight-bold">Chats</div>
        <div class="text-caption text-medium-emphasis">
          แชทลูกค้าจากทุกช่องทางในที่เดียว
          <v-chip size="x-small" variant="tonal" :color="wsConnected ? 'success' : 'warning'" class="ml-1">
            {{ wsConnected ? 'realtime' : 'อัปเดตทุก 20 วิ' }}
          </v-chip>
        </div>
      </div>
      <v-spacer />
      <v-btn icon="mdi-refresh" variant="text" :loading="loading" @click="fetchConversations" />
    </div>

    <v-card>
      <div class="chat-shell">
        <!-- Conversation list -->
        <div class="conv-pane" :class="{ 'conv-pane--hidden': mobile && activeConv }">
          <div class="conv-toolbar pa-3 pb-0">
            <v-text-field
              v-model="search" density="compact" hide-details clearable
              placeholder="ค้นหาชื่อ / ข้อความ..." prepend-inner-icon="mdi-magnify" class="mb-2"
            />
            <div class="d-flex ga-1 mb-2 flex-wrap">
              <v-chip
                v-for="p in platformFilters" :key="p.value" size="small" label
                :variant="platformFilter === p.value ? 'flat' : 'tonal'"
                :color="platformFilter === p.value ? 'primary' : undefined"
                @click="platformFilter = p.value"
              >{{ p.label }}</v-chip>
            </div>
            <v-tabs v-model="statusTab" density="compact" color="secondary" grow>
              <v-tab value="waiting" class="text-none px-1">
                รอตอบ
                <v-chip v-if="waitingCount" size="x-small" color="error" variant="flat" class="ml-1">{{ waitingCount }}</v-chip>
              </v-tab>
              <v-tab value="active" class="text-none px-1">กำลังคุย</v-tab>
              <v-tab value="done" class="text-none px-1">จบแล้ว</v-tab>
            </v-tabs>
            <v-divider />
          </div>
          <div v-if="filteredConversations.length" class="conv-list">
            <div
              v-for="conv in filteredConversations" :key="conv.id"
              class="conv-item pa-3"
              :class="{ 'conv-item--active': activeConv?.id === conv.id }"
              @click="openConversation(conv)"
            >
              <v-badge
                :model-value="conv.unread_count > 0" :content="conv.unread_count"
                color="error" offset-x="6" offset-y="6"
              >
                <v-avatar size="42" color="grey-lighten-3">
                  <v-img v-if="conv.avatar_url" :src="conv.avatar_url" cover />
                  <span v-else class="text-body-2 font-weight-bold">{{ (conv.display_name || '?')[0] }}</span>
                </v-avatar>
              </v-badge>
              <div class="conv-item-body ml-3">
                <div class="d-flex align-center">
                  <v-icon :icon="platformIcon(conv.platform)" size="14" :color="platformColor(conv.platform)" class="mr-1" />
                  <span class="font-weight-medium text-body-2 text-truncate">{{ conv.display_name }}</span>
                  <v-spacer />
                  <span class="text-caption text-medium-emphasis">{{ timeAgo(conv.last_message_at) }}</span>
                </div>
                <div class="text-caption text-medium-emphasis text-truncate">{{ conv.last_message_text || '—' }}</div>
                <div v-if="waitingLabel(conv) || (conv.tags && conv.tags.length)" class="d-flex ga-1 mt-1 flex-wrap">
                  <v-chip
                    v-if="waitingLabel(conv)" size="x-small" label variant="tonal"
                    :color="waitingUrgent(conv) ? 'error' : 'warning'" prepend-icon="mdi-clock-outline"
                  >{{ waitingLabel(conv) }}</v-chip>
                  <v-chip v-for="tag in conv.tags || []" :key="tag" size="x-small" label variant="tonal" color="secondary">{{ tag }}</v-chip>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-center pa-10 text-medium-emphasis">
            <template v-if="conversations.length">
              <v-icon :icon="statusTab === 'waiting' ? 'mdi-check-all' : 'mdi-filter-outline'" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">{{ statusTab === 'waiting' ? 'ตอบครบทุกคนแล้ว 🎉' : 'ไม่มีแชทในหมวดนี้' }}</div>
            </template>
            <template v-else>
              <v-icon icon="mdi-chat-outline" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2 mb-1">ยังไม่มีแชทเข้ามา</div>
              <div class="text-caption" style="max-width: 260px; margin: 0 auto;">
                ตั้งค่า webhook ของ LINE / Facebook / Instagram<br>มาที่ <code>{{ webhookURL }}</code>
              </div>
            </template>
          </div>
        </div>

        <!-- Thread -->
        <div class="thread-pane" :class="{ 'thread-pane--hidden': mobile && !activeConv }">
          <template v-if="activeConv">
            <!-- Thread header -->
            <div class="thread-header pa-3 d-flex align-center">
              <v-btn v-if="mobile" icon="mdi-arrow-left" size="small" variant="text" class="mr-1" @click="activeConv = null" />
              <v-avatar size="36" color="grey-lighten-3" class="mr-2">
                <v-img v-if="activeConv.avatar_url" :src="activeConv.avatar_url" cover />
                <span v-else class="text-body-2 font-weight-bold">{{ (activeConv.display_name || '?')[0] }}</span>
              </v-avatar>
              <div>
                <div class="font-weight-medium text-body-2">{{ activeConv.display_name }}</div>
                <div class="text-caption text-medium-emphasis d-flex align-center">
                  <v-icon :icon="platformIcon(activeConv.platform)" size="12" :color="platformColor(activeConv.platform)" class="mr-1" />
                  {{ activeConv.platform.toUpperCase() }}
                  <template v-if="activeConv.customer">
                    · ลูกค้า: {{ activeConv.customer.name }}
                  </template>
                </div>
              </div>
              <v-spacer />
              <div class="d-flex ga-1 align-center flex-wrap justify-end">
                <v-chip v-for="tag in activeConv.tags || []" :key="tag" size="x-small" label variant="tonal" color="secondary">{{ tag }}</v-chip>
                <v-btn icon="mdi-tag-outline" size="small" variant="text" title="ป้ายกำกับ" @click="openTagDialog" />
                <v-btn
                  size="small" variant="text" class="text-none"
                  prepend-icon="mdi-account-link-outline"
                  @click="openLinkDialog"
                >
                  {{ activeConv.customer ? 'เปลี่ยนลูกค้า' : 'ผูกลูกค้า' }}
                </v-btn>
                <v-btn
                  v-if="activeConv.status !== 'done'"
                  size="small" variant="tonal" color="success" class="text-none"
                  prepend-icon="mdi-check" :loading="statusSaving" @click="toggleStatus('done')"
                >จบงาน</v-btn>
                <v-btn
                  v-else
                  size="small" variant="tonal" color="warning" class="text-none"
                  prepend-icon="mdi-restore" :loading="statusSaving" @click="toggleStatus('open')"
                >เปิดใหม่</v-btn>
              </div>
            </div>
            <v-divider />

            <!-- Messages -->
            <div ref="threadEl" class="thread-messages pa-4">
              <div
                v-for="msg in messages" :key="msg.id"
                class="d-flex mb-2"
                :class="msg.direction === 'out' ? 'justify-end' : 'justify-start'"
              >
                <div class="bubble" :class="msg.direction === 'out' ? 'bubble--out' : 'bubble--in'">
                  <v-img
                    v-if="msg.type === 'image' && msg.image_url"
                    :src="msg.image_url" max-width="240" rounded="lg" class="mb-1" cover
                  />
                  <div v-if="msg.text" class="text-body-2" style="white-space: pre-wrap;">{{ msg.text }}</div>
                  <div class="bubble-time text-caption">{{ formatTime(msg.created_at) }}</div>
                </div>
              </div>
            </div>

            <!-- Composer -->
            <v-divider />
            <div class="pa-3 d-flex align-end ga-2">
              <v-menu location="top start">
                <template v-slot:activator="{ props }">
                  <v-btn v-bind="props" icon="mdi-flash-outline" variant="tonal" color="secondary" title="ข้อความสำเร็จรูป" />
                </template>
                <v-list density="compact" max-height="300" width="300">
                  <v-list-item
                    v-for="cr in cannedReplies" :key="cr.id"
                    :title="cr.title" :subtitle="cr.text"
                    @click="draft = draft ? draft + '\n' + cr.text : cr.text"
                  />
                  <v-list-item v-if="!cannedReplies.length" title="ยังไม่มีข้อความสำเร็จรูป" disabled />
                  <v-divider />
                  <v-list-item prepend-icon="mdi-cog-outline" title="จัดการข้อความสำเร็จรูป" @click="cannedDialog = true" />
                </v-list>
              </v-menu>
              <v-textarea
                v-model="draft" placeholder="พิมพ์ข้อความตอบกลับ..." rows="1" auto-grow max-rows="4"
                hide-details density="compact" @keydown.enter.exact.prevent="sendReply"
              />
              <v-btn
                color="primary" icon="mdi-send" :loading="sending"
                :disabled="!draft.trim()" @click="sendReply"
              />
            </div>
            <v-alert v-if="sendError" type="error" variant="tonal" density="compact" class="ma-3 mt-0">
              {{ sendError }}
            </v-alert>
          </template>
          <div v-else class="d-flex align-center justify-center fill-height text-medium-emphasis pa-10">
            <div class="text-center">
              <v-icon icon="mdi-forum-outline" size="48" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">เลือกแชทจากรายการด้านซ้าย</div>
            </div>
          </div>
        </div>
      </div>
    </v-card>

    <!-- Tags dialog -->
    <v-dialog v-model="tagDialog" max-width="440">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">ป้ายกำกับแชท</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-combobox
            v-model="tagDraft" :items="tagSuggestions" multiple chips closable-chips
            label="ป้ายกำกับ" hint="พิมพ์เองแล้วกด Enter เพื่อเพิ่มป้ายใหม่" persistent-hint
          />
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="tagDialog = false">ยกเลิก</v-btn>
          <v-btn color="primary" class="text-none px-6" :loading="tagSaving" @click="saveTags">บันทึก</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Canned replies manager -->
    <v-dialog v-model="cannedDialog" max-width="560">
      <v-card>
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <span class="text-h6 font-weight-bold">ข้อความสำเร็จรูป</span>
          <v-spacer />
          <v-btn size="small" variant="tonal" color="primary" class="text-none" prepend-icon="mdi-plus"
            @click="editCanned(null)">เพิ่มใหม่</v-btn>
        </v-card-title>
        <v-card-text class="px-5 pb-5">
          <template v-if="cannedForm">
            <v-text-field v-model="cannedForm.title" label="ชื่อ (เช่น ค่าส่ง, เลขบัญชี)" class="mb-1" />
            <v-textarea v-model="cannedForm.text" label="ข้อความ" rows="3" />
            <div class="d-flex justify-end ga-2">
              <v-btn variant="text" class="text-none" @click="cannedForm = null">ยกเลิก</v-btn>
              <v-btn color="primary" class="text-none" :loading="cannedSaving"
                :disabled="!cannedForm.title.trim() || !cannedForm.text.trim()" @click="saveCanned">บันทึก</v-btn>
            </div>
          </template>
          <v-list v-else density="comfortable">
            <v-list-item v-for="cr in cannedReplies" :key="cr.id" :title="cr.title" :subtitle="cr.text">
              <template v-slot:append>
                <v-btn icon="mdi-pencil-outline" size="x-small" variant="text" @click="editCanned(cr)" />
                <v-btn icon="mdi-delete-outline" size="x-small" variant="text" color="error" @click="deleteCanned(cr)" />
              </template>
            </v-list-item>
            <div v-if="!cannedReplies.length" class="text-center pa-6 text-medium-emphasis text-body-2">
              ยังไม่มีข้อความสำเร็จรูป — เพิ่มคำตอบที่พิมพ์บ่อย เช่น ค่าส่ง เลขบัญชี ไซส์ชาร์ต
            </div>
          </v-list>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Link customer dialog -->
    <v-dialog v-model="linkDialog" max-width="440">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">ผูกแชทกับลูกค้า</span>
        </v-card-title>
        <v-card-text class="px-5">
          <div class="text-caption text-medium-emphasis mb-3">
            ผูกแล้วจะเห็นชื่อลูกค้าในแชท และใช้สร้างออเดอร์ให้ถูกคนได้เร็วขึ้น
          </div>
          <v-autocomplete
            v-model="linkCustomerId"
            :items="customers"
            item-title="name"
            item-value="id"
            label="เลือกลูกค้า"
            clearable
          >
            <template v-slot:item="{ item, props }">
              <v-list-item v-bind="props" :subtitle="item.raw.phone || '-'" />
            </template>
          </v-autocomplete>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="linkDialog = false">ยกเลิก</v-btn>
          <v-btn color="primary" class="text-none px-6" :loading="linking" @click="saveLink">บันทึก</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, computed } from 'vue'
import { useDisplay } from 'vuetify'
import api from '@/services/api'

interface Conversation {
  id: number; platform: string; external_id: string;
  display_name: string; avatar_url: string;
  customer_id: number | null; customer?: { id: number; name: string } | null;
  unread_count: number; last_message_text: string; last_message_at: string;
  status: string; last_direction: string; waiting_since: string | null; tags: string[] | null;
}
interface CannedReply { id: number; title: string; text: string }
interface ChatMessage {
  id: number; conversation_id: number; direction: string;
  type: string; text: string; image_url: string; created_at: string;
}

const { mobile } = useDisplay()

const conversations = ref<Conversation[]>([])
const activeConv = ref<Conversation | null>(null)
const messages = ref<ChatMessage[]>([])
const loading = ref(false)
const draft = ref('')
const sending = ref(false)
const sendError = ref('')
const threadEl = ref<HTMLElement | null>(null)

const webhookURL = computed(() => `${window.location.origin}/api/webhooks/…`)

// ── Filters: status tabs + platform + search ──
const statusTab = ref('waiting')
const platformFilter = ref('all')
const search = ref('')

const platformFilters = [
  { label: 'ทั้งหมด', value: 'all' },
  { label: 'LINE', value: 'line' },
  { label: 'Facebook', value: 'facebook' },
  { label: 'IG', value: 'instagram' },
]

function isWaiting(c: Conversation) {
  return c.status !== 'done' && c.last_direction === 'in'
}

const waitingCount = computed(() => conversations.value.filter(isWaiting).length)

const filteredConversations = computed(() => {
  let list = conversations.value
  if (statusTab.value === 'waiting') list = list.filter(isWaiting)
  else if (statusTab.value === 'active') list = list.filter(c => c.status !== 'done' && !isWaiting(c))
  else list = list.filter(c => c.status === 'done')
  if (platformFilter.value !== 'all') list = list.filter(c => c.platform === platformFilter.value)
  const q = (search.value || '').trim().toLowerCase()
  if (q) {
    list = list.filter(c =>
      (c.display_name || '').toLowerCase().includes(q) ||
      (c.last_message_text || '').toLowerCase().includes(q) ||
      (c.customer?.name || '').toLowerCase().includes(q))
  }
  // Waiting tab: longest-waiting first so the queue works oldest-up.
  if (statusTab.value === 'waiting') {
    return [...list].sort((a, b) =>
      new Date(a.waiting_since || a.last_message_at).getTime() -
      new Date(b.waiting_since || b.last_message_at).getTime())
  }
  return list
})

function waitingLabel(c: Conversation): string {
  if (!isWaiting(c) || !c.waiting_since) return ''
  const mins = Math.floor((Date.now() - new Date(c.waiting_since).getTime()) / 60000)
  if (mins < 1) return 'รอเมื่อครู่'
  if (mins < 60) return `รอ ${mins} นาที`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `รอ ${hours} ชม.`
  return `รอ ${Math.floor(hours / 24)} วัน`
}
function waitingUrgent(c: Conversation): boolean {
  if (!c.waiting_since) return false
  return Date.now() - new Date(c.waiting_since).getTime() > 60 * 60000
}

// ── Status (จบงาน / เปิดใหม่) ──
const statusSaving = ref(false)
async function toggleStatus(status: 'open' | 'done') {
  if (!activeConv.value) return
  statusSaving.value = true
  try {
    const { data } = await api.put(`/chats/${activeConv.value.id}/status`, { status })
    activeConv.value = data
    fetchConversations()
  } finally {
    statusSaving.value = false
  }
}

// ── Tags ──
const tagDialog = ref(false)
const tagDraft = ref<string[]>([])
const tagSaving = ref(false)
const tagSuggestions = ['รอโอน', 'โอนแล้ว', 'CF', 'รอของเข้า', 'ส่งแล้ว', 'ถามเฉยๆ']

function openTagDialog() {
  tagDraft.value = [...(activeConv.value?.tags || [])]
  tagDialog.value = true
}
async function saveTags() {
  if (!activeConv.value) return
  tagSaving.value = true
  try {
    const { data } = await api.put(`/chats/${activeConv.value.id}/tags`, { tags: tagDraft.value })
    activeConv.value = data
    tagDialog.value = false
    fetchConversations()
  } finally {
    tagSaving.value = false
  }
}

// ── Canned replies ──
const cannedReplies = ref<CannedReply[]>([])
const cannedDialog = ref(false)
const cannedForm = ref<{ id: number | null; title: string; text: string } | null>(null)
const cannedSaving = ref(false)

async function fetchCanned() {
  const { data } = await api.get('/canned-replies')
  cannedReplies.value = data || []
}
function editCanned(cr: CannedReply | null) {
  cannedForm.value = cr ? { ...cr } : { id: null, title: '', text: '' }
}
async function saveCanned() {
  if (!cannedForm.value) return
  cannedSaving.value = true
  try {
    if (cannedForm.value.id) {
      await api.put(`/canned-replies/${cannedForm.value.id}`, cannedForm.value)
    } else {
      await api.post('/canned-replies', cannedForm.value)
    }
    cannedForm.value = null
    await fetchCanned()
  } finally {
    cannedSaving.value = false
  }
}
async function deleteCanned(cr: CannedReply) {
  await api.delete(`/canned-replies/${cr.id}`)
  await fetchCanned()
}

async function fetchConversations() {
  loading.value = true
  try {
    const { data } = await api.get('/chats')
    conversations.value = data || []
    // Keep the open thread's header info fresh (name/customer/unread).
    if (activeConv.value) {
      const updated = conversations.value.find(c => c.id === activeConv.value!.id)
      if (updated) activeConv.value = updated
    }
  } finally {
    loading.value = false
  }
}

async function openConversation(conv: Conversation) {
  activeConv.value = conv
  sendError.value = ''
  const { data } = await api.get(`/chats/${conv.id}/messages`)
  messages.value = data.messages || []
  scrollToBottom()
  if (conv.unread_count > 0) {
    conv.unread_count = 0
    api.post(`/chats/${conv.id}/read`)
  }
}

async function sendReply() {
  const text = draft.value.trim()
  if (!text || !activeConv.value) return
  sending.value = true
  sendError.value = ''
  try {
    const { data } = await api.post(`/chats/${activeConv.value.id}/reply`, { text })
    messages.value.push(data)
    draft.value = ''
    scrollToBottom()
    fetchConversations()
  } catch (err: any) {
    sendError.value = err.response?.data?.error || 'ส่งไม่สำเร็จ'
  } finally {
    sending.value = false
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (threadEl.value) threadEl.value.scrollTop = threadEl.value.scrollHeight
  })
}

// ── Link customer ──
const linkDialog = ref(false)
const linkCustomerId = ref<number | null>(null)
const linking = ref(false)
const customers = ref<any[]>([])

async function openLinkDialog() {
  if (!customers.value.length) {
    const { data } = await api.get('/customers')
    customers.value = data || []
  }
  linkCustomerId.value = activeConv.value?.customer_id || null
  linkDialog.value = true
}

async function saveLink() {
  if (!activeConv.value) return
  linking.value = true
  try {
    const { data } = await api.put(`/chats/${activeConv.value.id}/customer`, {
      customer_id: linkCustomerId.value || 0,
    })
    activeConv.value = data
    linkDialog.value = false
    fetchConversations()
  } finally {
    linking.value = false
  }
}

// ── Realtime: WebSocket with polling fallback ──
const wsConnected = ref(false)
let ws: WebSocket | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let closed = false

function connectWS() {
  const token = localStorage.getItem('token')
  if (!token) return
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  ws = new WebSocket(`${proto}://${window.location.host}/api/ws/chat?token=${token}`)
  ws.onopen = () => { wsConnected.value = true }
  ws.onmessage = (ev) => {
    try {
      const payload = JSON.parse(ev.data)
      if (payload.type !== 'message') return
      if (activeConv.value && payload.conversation_id === activeConv.value.id) {
        // Skip our own replies — they're appended locally on send.
        if (!messages.value.some(m => m.id === payload.message.id)) {
          messages.value.push(payload.message)
          scrollToBottom()
        }
        if (payload.message.direction === 'in') {
          api.post(`/chats/${activeConv.value.id}/read`)
        }
      }
      fetchConversations()
    } catch {}
  }
  ws.onclose = () => {
    wsConnected.value = false
    if (!closed) reconnectTimer = setTimeout(connectWS, 5000)
  }
  ws.onerror = () => ws?.close()
}

// ── Helpers ──
function platformIcon(p: string) {
  return ({ line: 'mdi-chat', facebook: 'mdi-facebook-messenger', instagram: 'mdi-instagram' } as Record<string, string>)[p] || 'mdi-chat-outline'
}
function platformColor(p: string) {
  return ({ line: '#008300', facebook: '#2a78d6', instagram: '#4a3aa7' } as Record<string, string>)[p] || 'grey'
}
function timeAgo(iso: string) {
  const mins = Math.floor((Date.now() - new Date(iso).getTime()) / 60000)
  if (mins < 1) return 'เมื่อครู่'
  if (mins < 60) return `${mins} นาที`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} ชม.`
  return `${Math.floor(hours / 24)} วัน`
}
function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })
}

onMounted(() => {
  fetchConversations()
  fetchCanned()
  connectWS()
  pollTimer = setInterval(() => {
    if (!wsConnected.value) fetchConversations()
  }, 20_000)
})

onUnmounted(() => {
  closed = true
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (pollTimer) clearInterval(pollTimer)
  ws?.close()
})
</script>

<style scoped>
.chat-shell {
  display: flex;
  height: calc(100vh - 210px);
  min-height: 420px;
}
.conv-pane {
  width: 320px;
  min-width: 320px;
  border-right: 1px solid #E5E7EB;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.conv-list {
  flex: 1;
  overflow-y: auto;
}
.thread-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.conv-item {
  display: flex;
  align-items: center;
  cursor: pointer;
  border-bottom: 1px solid #F1F3F5;
  transition: background 0.15s;
}
.conv-item:hover {
  background: #F6F7F9;
}
.conv-item--active {
  background: rgba(196, 162, 77, 0.10);
}
.conv-item-body {
  flex: 1;
  min-width: 0;
}
.thread-messages {
  flex: 1;
  overflow-y: auto;
  background: #F6F7F9;
}
.bubble {
  max-width: 70%;
  padding: 8px 12px;
  border-radius: 14px;
}
.bubble--in {
  background: #FFFFFF;
  border: 1px solid #E5E7EB;
  border-top-left-radius: 4px;
}
.bubble--out {
  background: #1A1714;
  color: #fff;
  border-top-right-radius: 4px;
}
.bubble--out .bubble-time {
  color: rgba(255, 255, 255, 0.55);
}
.bubble-time {
  font-size: 10px !important;
  color: #9CA3AF;
  text-align: right;
  margin-top: 2px;
}
@media (max-width: 960px) {
  .conv-pane {
    width: 100%;
    min-width: 0;
    border-right: none;
  }
  .conv-pane--hidden {
    display: none;
  }
  .thread-pane--hidden {
    display: none;
  }
}
</style>
