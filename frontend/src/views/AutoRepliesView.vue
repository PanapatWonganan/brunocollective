<template>
  <div>
    <div class="d-flex align-center mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Auto Reply</div>
        <div class="text-caption text-medium-emphasis">
          ตอบคอมเมนต์ Facebook / Instagram อัตโนมัติตาม keyword
        </div>
      </div>
      <v-spacer />
      <v-btn color="primary" prepend-icon="mdi-plus" class="text-none" @click="openDialog()">
        เพิ่มกฎใหม่
      </v-btn>
    </div>

    <!-- Rules -->
    <v-card class="mb-4">
      <v-card-text class="pa-5">
        <v-data-table
          :headers="ruleHeaders"
          :items="rules"
          :loading="loading"
          items-per-page="10"
          class="rounded-lg"
        >
          <template v-slot:item.name="{ item }">
            <div class="font-weight-medium">{{ item.name }}</div>
            <div class="text-caption text-medium-emphasis">{{ actionSummary(item) }}</div>
          </template>
          <template v-slot:item.platform="{ item }">
            <v-chip size="small" variant="tonal" label :color="platformColor(item.platform)">
              {{ platformLabel(item.platform) }}
            </v-chip>
          </template>
          <template v-slot:item.keywords="{ item }">
            <div class="d-flex flex-wrap ga-1 py-1">
              <v-chip v-for="kw in (item.keywords || []).slice(0, 4)" :key="kw" size="x-small" label variant="tonal" color="secondary">
                {{ kw }}
              </v-chip>
              <v-chip v-if="(item.keywords || []).length > 4" size="x-small" label variant="tonal">
                +{{ item.keywords.length - 4 }}
              </v-chip>
              <span v-if="!(item.keywords || []).length" class="text-caption text-medium-emphasis">ทุกคอมเมนต์</span>
            </div>
          </template>
          <template v-slot:item.usage_count="{ item }">
            {{ item.usage_count }} ครั้ง
          </template>
          <template v-slot:item.enabled="{ item }">
            <v-switch
              :model-value="item.enabled" color="success" hide-details density="compact"
              @update:model-value="toggleRule(item)"
            />
          </template>
          <template v-slot:item.actions="{ item }">
            <v-btn icon="mdi-pencil-outline" size="small" variant="text" color="primary" @click="openDialog(item)" />
            <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" @click="confirmDelete(item)" />
          </template>
          <template v-slot:no-data>
            <div class="text-center pa-8 text-medium-emphasis">
              <v-icon icon="mdi-robot-outline" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2 mb-1">ยังไม่มีกฎ auto-reply</div>
              <div class="text-caption">เช่น ลูกค้าคอมเมนต์ "สนใจ" → ตอบกลับ + ส่งราคาทาง DM</div>
            </div>
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Logs -->
    <v-card>
      <v-card-text class="pa-5">
        <div class="d-flex align-center mb-3">
          <div>
            <div class="text-subtitle-1 font-weight-bold" style="color: #111827;">ประวัติการทำงาน</div>
            <div class="text-caption" style="color: #6B7280;">100 รายการล่าสุด</div>
          </div>
          <v-spacer />
          <v-btn icon="mdi-refresh" size="small" variant="text" @click="fetchLogs" />
        </div>
        <v-table v-if="logs.length" density="comfortable" class="rounded-lg">
          <thead>
            <tr>
              <th class="table-header">เวลา</th>
              <th class="table-header">แพลตฟอร์ม</th>
              <th class="table-header">คอมเมนต์</th>
              <th class="table-header">กฎที่ใช้</th>
              <th class="table-header">การทำงาน</th>
              <th class="table-header text-center">ผล</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="lg in logs" :key="lg.id">
              <td class="text-caption text-no-wrap">{{ formatDate(lg.created_at) }}</td>
              <td>
                <v-chip size="x-small" variant="tonal" label :color="platformColor(lg.platform)">
                  {{ platformLabel(lg.platform) }}
                </v-chip>
              </td>
              <td style="max-width: 280px;">
                <div class="font-weight-medium text-body-2">{{ lg.from_name || '-' }}</div>
                <div class="text-caption text-medium-emphasis text-truncate">{{ lg.comment_text }}</div>
              </td>
              <td class="text-body-2">{{ lg.rule_name }}</td>
              <td>
                <div class="d-flex ga-1 flex-wrap">
                  <v-chip v-for="a in (lg.actions || '').split(',').filter(Boolean)" :key="a" size="x-small" label variant="tonal">
                    {{ actionLabel(a) }}
                  </v-chip>
                </div>
              </td>
              <td class="text-center">
                <v-tooltip :text="lg.error || 'สำเร็จ'" location="top">
                  <template v-slot:activator="{ props }">
                    <v-chip v-bind="props" size="x-small" label variant="tonal"
                      :color="lg.status === 'success' ? 'success' : lg.status === 'partial' ? 'warning' : 'error'">
                      {{ lg.status === 'success' ? 'สำเร็จ' : lg.status === 'partial' ? 'บางส่วน' : 'ล้มเหลว' }}
                    </v-chip>
                  </template>
                </v-tooltip>
              </td>
            </tr>
          </tbody>
        </v-table>
        <div v-else class="text-center pa-8 text-medium-emphasis">
          <v-icon icon="mdi-history" size="32" class="mb-2" color="grey-lighten-1" />
          <div class="text-body-2">ยังไม่มีประวัติ — จะบันทึกทุกครั้งที่บอทตอบคอมเมนต์</div>
        </div>
      </v-card-text>
    </v-card>

    <!-- Rule editor -->
    <v-dialog v-model="dialog" max-width="640" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">{{ editingRule ? 'แก้ไขกฎ' : 'เพิ่มกฎใหม่' }}</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-text-field v-model="form.name" label="ชื่อกฎ (เช่น ตอบคนถามราคา)" class="mb-1" />
          <div class="d-flex ga-2">
            <v-select
              v-model="form.platform" :items="platformOptions" item-title="label" item-value="value"
              label="แพลตฟอร์ม" class="mb-1"
            />
            <v-text-field
              v-model.number="form.priority" label="ลำดับ" type="number"
              hint="น้อย = เช็คก่อน" persistent-hint style="max-width: 120px;"
            />
          </div>
          <v-combobox
            v-model="form.keywords" :items="keywordSuggestions" multiple chips closable-chips
            label="Keywords ที่ trigger" hint="พิมพ์แล้วกด Enter — เว้นว่าง = ทุกคอมเมนต์" persistent-hint class="mb-3"
          />

          <div class="text-subtitle-2 font-weight-medium mb-2">การทำงานเมื่อเจอ keyword (เลือกได้หลายอย่าง)</div>
          <v-textarea
            v-model="form.reply_text" label="1) ตอบใต้คอมเมนต์ (เว้นว่าง = ไม่ตอบ)" rows="2" class="mb-1"
            hint="ใช้ {name} แทนชื่อลูกค้าได้ เช่น ขอบคุณคุณ {name} ครับ ทักแชทมาได้เลย" persistent-hint
          />
          <v-textarea
            v-model="form.private_reply_text" label="2) ส่งข้อความส่วนตัว DM (เว้นว่าง = ไม่ส่ง)" rows="2" class="mb-1"
            hint="ส่งได้ 1 ครั้งต่อคอมเมนต์ — เหมาะกับส่งราคา/ลิงก์สั่งซื้อ" persistent-hint
          />
          <v-checkbox v-model="form.hide_comment" hide-details density="compact"
            label="3) ซ่อนคอมเมนต์จากคนอื่น (เหมาะกับโพสต์ CF กันคนเห็นราคา/ตัดหน้า)" />

          <v-alert v-if="formError" type="error" variant="tonal" density="compact" class="mt-2">
            {{ formError }}
          </v-alert>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="dialog = false">ยกเลิก</v-btn>
          <v-btn color="primary" class="text-none px-6" :loading="saving" @click="saveRule">
            {{ editingRule ? 'บันทึก' : 'สร้างกฎ' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete confirm -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card class="text-center pa-2">
        <v-card-text class="pt-5">
          <v-avatar color="error" variant="tonal" size="56" class="mb-4">
            <v-icon icon="mdi-delete-outline" size="28" />
          </v-avatar>
          <div class="text-h6 font-weight-bold mb-2">ลบกฎนี้?</div>
          <div class="text-body-2 text-medium-emphasis">
            ลบ "<strong>{{ deletingRule?.name }}</strong>" แล้วบอทจะหยุดตอบตามกฎนี้ทันที
          </div>
        </v-card-text>
        <v-card-actions class="justify-center pb-5">
          <v-btn variant="text" class="text-none" @click="deleteDialog = false">ยกเลิก</v-btn>
          <v-btn color="error" class="text-none px-6" :loading="saving" @click="deleteRule">ลบ</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/services/api'

interface Rule {
  id?: number; name: string; platform: string; keywords: string[];
  enabled: boolean; priority: number;
  reply_text: string; private_reply_text: string; hide_comment: boolean;
  usage_count?: number;
}
interface Log {
  id: number; rule_name: string; platform: string; comment_id: string;
  from_name: string; comment_text: string; actions: string;
  status: string; error: string; created_at: string;
}

const rules = ref<Rule[]>([])
const logs = ref<Log[]>([])
const loading = ref(false)
const dialog = ref(false)
const deleteDialog = ref(false)
const saving = ref(false)
const editingRule = ref<Rule | null>(null)
const deletingRule = ref<Rule | null>(null)
const formError = ref('')

const ruleHeaders = [
  { title: 'กฎ', key: 'name' },
  { title: 'แพลตฟอร์ม', key: 'platform', align: 'center' as const },
  { title: 'Keywords', key: 'keywords', sortable: false },
  { title: 'ใช้ไปแล้ว', key: 'usage_count', align: 'center' as const },
  { title: 'เปิดใช้', key: 'enabled', align: 'center' as const, width: '90px' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '100px' },
]

const platformOptions = [
  { label: 'ทุกแพลตฟอร์ม', value: 'all' },
  { label: 'Facebook', value: 'facebook' },
  { label: 'Instagram', value: 'instagram' },
]

const keywordSuggestions = ['สนใจ', 'ราคา', 'เท่าไหร่', 'เท่าไร', 'cf', 'CF', 'ขอราคา', 'สั่งซื้อ', 'มีไซส์', 'inbox']

const emptyForm = (): Rule => ({
  name: '', platform: 'all', keywords: [], enabled: true, priority: 100,
  reply_text: '', private_reply_text: '', hide_comment: false,
})
const form = ref<Rule>(emptyForm())

function platformLabel(p: string) {
  return ({ all: 'ทั้งหมด', facebook: 'Facebook', instagram: 'Instagram' } as Record<string, string>)[p] || p
}
function platformColor(p: string) {
  return ({ facebook: 'info', instagram: 'secondary', all: 'default' } as Record<string, string>)[p] || 'default'
}
function actionLabel(a: string) {
  return ({ reply: 'ตอบคอมเมนต์', private_reply: 'ส่ง DM', hide: 'ซ่อน' } as Record<string, string>)[a] || a
}
function actionSummary(r: Rule) {
  const parts: string[] = []
  if (r.reply_text) parts.push('ตอบคอมเมนต์')
  if (r.private_reply_text) parts.push('ส่ง DM')
  if (r.hide_comment) parts.push('ซ่อน')
  return parts.join(' + ')
}
function formatDate(d: string) {
  return new Date(d).toLocaleDateString('th-TH', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

async function fetchRules() {
  loading.value = true
  try {
    const { data } = await api.get('/auto-replies')
    rules.value = data || []
  } finally {
    loading.value = false
  }
}
async function fetchLogs() {
  const { data } = await api.get('/auto-replies/logs')
  logs.value = data || []
}

function openDialog(rule?: Rule) {
  editingRule.value = rule || null
  form.value = rule ? { ...rule, keywords: [...(rule.keywords || [])] } : emptyForm()
  formError.value = ''
  dialog.value = true
}

async function saveRule() {
  saving.value = true
  formError.value = ''
  try {
    if (editingRule.value?.id) {
      await api.put(`/auto-replies/${editingRule.value.id}`, form.value)
    } else {
      await api.post('/auto-replies', form.value)
    }
    dialog.value = false
    await fetchRules()
  } catch (err: any) {
    formError.value = err.response?.data?.error || 'บันทึกไม่สำเร็จ'
  } finally {
    saving.value = false
  }
}

async function toggleRule(rule: Rule) {
  await api.post(`/auto-replies/${rule.id}/toggle`)
  await fetchRules()
}

function confirmDelete(rule: Rule) {
  deletingRule.value = rule
  deleteDialog.value = true
}
async function deleteRule() {
  saving.value = true
  try {
    await api.delete(`/auto-replies/${deletingRule.value?.id}`)
    deleteDialog.value = false
    await fetchRules()
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchRules()
  fetchLogs()
})
</script>

<style scoped>
.table-header {
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: #6B7280 !important;
}
</style>
