<template>
  <div>
    <div class="d-flex align-center mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Site Images</div>
        <div class="text-caption text-medium-emphasis">
          Manage the hero, lookbook and journal images shown on the public storefront home page
        </div>
      </div>
    </div>

    <v-alert
      type="info" variant="tonal" density="comfortable" class="mb-6"
      icon="mdi-information-outline"
    >
      Slots left empty fall back to the storefront's built-in default images. Recommended: large landscape
      photos (at least 1200px wide).
    </v-alert>

    <template v-for="group in groups" :key="group.title">
      <div class="text-overline text-medium-emphasis mb-2 mt-2">{{ group.title }}</div>
      <v-row class="mb-4">
        <v-col v-for="slot in group.slots" :key="slot.key" cols="12" sm="6" md="4">
          <v-card class="slot-card h-100">
            <div class="slot-preview">
              <v-img
                v-if="bySlot[slot.key]?.image_url"
                :src="bySlot[slot.key].image_url"
                height="180" cover
              />
              <div v-else class="slot-empty">
                <v-icon icon="mdi-image-off-outline" size="32" color="grey" />
                <div class="text-caption text-medium-emphasis mt-1">Using default</div>
              </div>
            </div>
            <v-card-text class="pa-4">
              <div class="font-weight-medium mb-1">{{ slot.label }}</div>
              <div class="text-caption text-medium-emphasis mb-3">{{ slot.hint }}</div>

              <v-file-input
                :model-value="files[slot.key]"
                @update:model-value="(v: any) => onFileChange(slot.key, v)"
                label="Replace image"
                prepend-icon="mdi-camera"
                accept="image/*"
                density="compact"
                hide-details
                class="mb-3"
                :loading="uploading === slot.key"
              />

              <v-text-field
                v-model="captions[slot.key].caption_a"
                :label="slot.capALabel"
                density="compact"
                hide-details
                class="mb-2"
              />
              <v-text-field
                v-model="captions[slot.key].caption_b"
                :label="slot.capBLabel"
                density="compact"
                hide-details
                class="mb-3"
              />
              <v-btn
                size="small" color="primary" variant="tonal" class="text-none"
                :loading="savingCaptions === slot.key"
                @click="saveCaptions(slot.key)"
              >
                Save captions
              </v-btn>
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </template>

    <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="2500">
      {{ snackbar.text }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import api from '@/services/api'

interface SiteImage {
  key: string
  image_url: string
  caption_a: string
  caption_b: string
}

// Slot definitions mirror the storefront components. Labels/captions are
// purely for the admin UI; keys must match backend models.SiteImageSlots.
const groups = [
  {
    title: 'Hero',
    slots: [
      { key: 'hero', label: 'Hero background', hint: 'Full-width image at the top of the home page',
        capALabel: 'Caption A (unused)', capBLabel: 'Caption B (unused)' },
    ],
  },
  {
    title: 'Lookbook (6 tiles)',
    slots: Array.from({ length: 6 }, (_, i) => ({
      key: `lookbook_${i + 1}`,
      label: `Lookbook tile ${i + 1}`,
      hint: 'Editorial photo in the lookbook grid',
      capALabel: 'Title (e.g. "01 — The Reading Room")',
      capBLabel: 'Subtitle (e.g. "Milano, MMXXVI")',
    })),
  },
  {
    title: 'Journal (3 entries)',
    slots: Array.from({ length: 3 }, (_, i) => ({
      key: `journal_${i + 1}`,
      label: `Journal entry ${i + 1}`,
      hint: 'Cover photo for the journal article',
      capALabel: 'Tag (e.g. "Essay — N° 17")',
      capBLabel: 'Read time (e.g. "8 min")',
    })),
  },
]

const bySlot = ref<Record<string, SiteImage>>({})
const files = reactive<Record<string, File[]>>({})
const captions = reactive<Record<string, { caption_a: string; caption_b: string }>>({})
const uploading = ref<string | null>(null)
const savingCaptions = ref<string | null>(null)
const snackbar = reactive({ show: false, text: '', color: 'success' })

// Initialise caption state for every slot so v-model bindings are always defined.
for (const g of groups) for (const s of g.slots) {
  captions[s.key] = { caption_a: '', caption_b: '' }
}

function notify(text: string, color = 'success') {
  snackbar.text = text
  snackbar.color = color
  snackbar.show = true
}

async function fetchAll() {
  const { data } = await api.get<SiteImage[]>('/site-images')
  const map: Record<string, SiteImage> = {}
  for (const img of data || []) {
    map[img.key] = img
    captions[img.key] = { caption_a: img.caption_a || '', caption_b: img.caption_b || '' }
  }
  bySlot.value = map
}

async function onFileChange(key: string, value: File[] | File | null) {
  const arr = Array.isArray(value) ? value : value ? [value] : []
  files[key] = arr
  if (!arr.length) return
  uploading.value = key
  try {
    const fd = new FormData()
    fd.append('image', arr[0])
    const { data } = await api.post<SiteImage>(`/site-images/${key}/image`, fd)
    bySlot.value = { ...bySlot.value, [key]: data }
    files[key] = []
    notify('Image updated')
  } catch {
    notify('Upload failed', 'error')
  } finally {
    uploading.value = null
  }
}

async function saveCaptions(key: string) {
  savingCaptions.value = key
  try {
    const { data } = await api.put<SiteImage>(`/site-images/${key}`, captions[key])
    bySlot.value = { ...bySlot.value, [key]: data }
    notify('Captions saved')
  } catch {
    notify('Save failed', 'error')
  } finally {
    savingCaptions.value = null
  }
}

onMounted(fetchAll)
</script>

<style scoped>
.slot-card {
  border: 1px solid #E8E2D9 !important;
  display: flex;
  flex-direction: column;
}
.slot-preview {
  background: #F2ECE3;
}
.slot-empty {
  height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
</style>
