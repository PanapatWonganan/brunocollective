<template>
  <div>
    <div class="d-flex flex-wrap align-center ga-3 mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Sale Pages</div>
        <div class="text-caption text-medium-emphasis">Funnel-style landing pages with order bump — served at /s/&lt;slug&gt;</div>
      </div>
      <v-spacer />
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreate" class="text-none">
        New Sale Page
      </v-btn>
    </div>

    <v-card>
      <v-card-text class="pa-5">
        <v-data-table
          :headers="headers"
          :items="pages"
          :loading="loading"
          items-per-page="10"
          class="rounded-lg"
        >
          <template v-slot:item.title="{ item }">
            <div class="font-weight-bold">{{ item.title }}</div>
            <div class="text-caption text-medium-emphasis">
              /s/{{ item.slug }}
              <v-btn icon="mdi-content-copy" size="x-small" variant="text" density="compact"
                @click="copyLink(item)" />
            </div>
          </template>
          <template v-slot:item.offer="{ item }">
            <div class="font-weight-medium">{{ item.product?.name || '-' }}</div>
            <div class="text-caption">
              <template v-if="item.offer_price != null">
                <span class="text-decoration-line-through text-medium-emphasis mr-1">{{ formatCurrency(item.product?.price || 0) }}</span>
                <span class="text-secondary font-weight-bold">{{ formatCurrency(item.offer_price) }}</span>
              </template>
              <template v-else>{{ formatCurrency(item.product?.price || 0) }}</template>
            </div>
          </template>
          <template v-slot:item.status="{ item }">
            <v-chip :color="item.status === 'published' ? 'success' : 'grey'" size="small" variant="tonal" label class="text-capitalize">
              {{ item.status }}
            </v-chip>
          </template>
          <template v-slot:item.stats="{ item }">
            <div class="text-caption">
              <div>{{ item.views }} views · {{ item.orders_count }} orders</div>
              <div class="text-medium-emphasis">
                conv {{ conversionRate(item) }}
                <template v-if="item.bump_enabled"> · bump {{ bumpRate(item) }}</template>
              </div>
            </div>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-tooltip text="Open page" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-open-in-new" size="small" variant="text"
                  :href="`/s/${item.slug}?preview=1`" target="_blank" />
              </template>
            </v-tooltip>
            <v-tooltip :text="item.status === 'published' ? 'Unpublish' : 'Publish'" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" :icon="item.status === 'published' ? 'mdi-eye-off-outline' : 'mdi-rocket-launch-outline'"
                  size="small" variant="text" color="secondary" @click="togglePublish(item)" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-pencil-outline" size="small" variant="text" color="primary" @click="openEdit(item)" />
            <v-tooltip text="Duplicate" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-content-duplicate" size="small" variant="text" @click="duplicatePage(item)" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" @click="confirmDelete(item)" />
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Editor -->
    <v-dialog v-model="editorDialog" max-width="880" persistent scrollable :fullscreen="mobile">
      <v-card>
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <span class="text-h6 font-weight-bold">{{ editingId ? 'Edit Sale Page' : 'New Sale Page' }}</span>
          <v-spacer />
          <v-btn icon="mdi-close" size="small" variant="text" @click="editorDialog = false" />
        </v-card-title>

        <v-tabs v-model="editorTab" color="secondary" class="px-5">
          <v-tab value="offer" class="text-none">Offer &amp; Checkout</v-tab>
          <v-tab value="sections" class="text-none">Page Sections</v-tab>
        </v-tabs>
        <v-divider />

        <v-card-text class="px-5" style="max-height: 65vh;">
          <v-window v-model="editorTab">
            <!-- ============ OFFER TAB ============ -->
            <v-window-item value="offer">
              <v-row dense class="mt-2">
                <v-col cols="12" sm="7">
                  <v-text-field v-model="form.title" label="Page Title" :rules="[v => !!v?.trim() || 'Required']" />
                </v-col>
                <v-col cols="12" sm="5">
                  <v-text-field v-model="form.slug" label="Slug (URL)" prefix="/s/" hint="lowercase-with-dashes" />
                </v-col>

                <v-col cols="12" sm="7">
                  <v-select
                    v-model="form.product_id"
                    :items="products"
                    item-title="name"
                    item-value="id"
                    label="Main Product"
                  >
                    <template v-slot:item="{ item: prod, props }">
                      <v-list-item v-bind="props" :subtitle="`Stock: ${prod.raw.total_stock ?? prod.raw.stock} | ${formatCurrency(prod.raw.price)}`" />
                    </template>
                  </v-select>
                </v-col>
                <v-col cols="12" sm="5">
                  <v-text-field
                    v-model.number="offerPriceInput"
                    label="Offer Price (THB)"
                    type="number" min="0"
                    :hint="selectedProduct ? `Catalog price ${formatCurrency(selectedProduct.price)} — leave 0 to use it` : 'Leave 0 to sell at catalog price'"
                    persistent-hint
                  />
                </v-col>

                <v-col cols="12"><v-divider class="my-2" /></v-col>

                <v-col cols="12">
                  <div class="d-flex align-center">
                    <div>
                      <div class="text-subtitle-2 font-weight-bold">💰 Order Bump</div>
                      <div class="text-caption text-medium-emphasis">Add-on checkbox on the order form (quantity 1, special price)</div>
                    </div>
                    <v-spacer />
                    <v-switch v-model="form.bump_enabled" color="secondary" hide-details />
                  </div>
                </v-col>
                <template v-if="form.bump_enabled">
                  <v-col cols="12" sm="7">
                    <v-select
                      v-model="form.bump_product_id"
                      :items="products"
                      item-title="name"
                      item-value="id"
                      label="Bump Product"
                    />
                  </v-col>
                  <v-col cols="12" sm="5">
                    <v-text-field v-model.number="form.bump_price" label="Bump Price (THB)" type="number" min="0"
                      :hint="bumpProduct ? `Catalog price ${formatCurrency(bumpProduct.price)}` : ''" persistent-hint />
                  </v-col>
                  <v-col cols="12">
                    <v-text-field v-model="form.bump_headline" label="Bump Headline"
                      placeholder="เพิ่มหมวก Bruno ในราคาพิเศษ — เฉพาะออเดอร์นี้เท่านั้น" />
                  </v-col>
                  <v-col cols="12">
                    <v-textarea v-model="form.bump_description" label="Bump Description" rows="2"
                      placeholder="แมตช์กับเสื้อในออเดอร์นี้ ปกติ 490฿" />
                  </v-col>
                </template>

                <v-col cols="12"><v-divider class="my-2" /></v-col>

                <v-col cols="12" sm="6">
                  <v-text-field
                    v-model="countdownInput"
                    label="Countdown Ends (optional)"
                    type="datetime-local"
                    clearable
                    hint="Real deadline — orders are rejected after it passes"
                    persistent-hint
                  />
                </v-col>
                <v-col cols="12" sm="3">
                  <v-switch v-model="form.show_stock" label="Show real stock" color="secondary" hide-details />
                </v-col>
                <v-col cols="12" sm="3">
                  <v-switch v-model="form.allow_coupon" label="Accept coupons" color="secondary" hide-details />
                </v-col>
              </v-row>
            </v-window-item>

            <!-- ============ SECTIONS TAB ============ -->
            <v-window-item value="sections">
              <v-alert type="info" variant="tonal" density="compact" class="my-3">
                เปิดสวิตช์ท้ายแถวเพื่อใช้บล็อกนั้น แล้วกดแถวเพื่อกรอกเนื้อหา — เช่น อยากใส่รีวิวลูกค้า ให้เปิด
                <strong>Social Proof — testimonials</strong> · ถ้าไม่ใส่รูปเอง ระบบจะดึงรูปสินค้ามาแสดงให้อัตโนมัติ
                · ฟอร์มสั่งซื้ออยู่ท้ายเพจเสมอ
              </v-alert>
              <v-expansion-panels variant="accordion" class="mb-3">
                <v-expansion-panel v-for="(section, i) in form.sections" :key="section.type">
                  <v-expansion-panel-title>
                    <div class="d-flex align-center w-100">
                      <v-icon :icon="sectionMeta[section.type]?.icon || 'mdi-square-outline'" size="18" class="mr-3"
                        :color="section.enabled ? 'secondary' : 'grey'" />
                      <span :class="section.enabled ? 'font-weight-medium' : 'text-medium-emphasis'">
                        {{ sectionMeta[section.type]?.label || section.type }}
                      </span>
                      <v-spacer />
                      <v-btn icon="mdi-arrow-up" size="x-small" variant="text" :disabled="i === 0"
                        @click.stop="moveSection(i, -1)" />
                      <v-btn icon="mdi-arrow-down" size="x-small" variant="text" :disabled="i === form.sections.length - 1"
                        @click.stop="moveSection(i, 1)" />
                      <v-switch v-model="section.enabled" color="secondary" hide-details density="compact"
                        class="ml-2 flex-grow-0" @click.stop />
                    </div>
                  </v-expansion-panel-title>
                  <v-expansion-panel-text>
                    <!-- HERO -->
                    <template v-if="section.type === 'hero'">
                      <v-text-field v-model="section.data.kicker" label="Kicker (small top label)" placeholder="Limited Release" />
                      <v-textarea v-model="section.data.headline" label="Headline" rows="2"
                        placeholder="เสื้อที่คุณจะใส่ไปอีกสิบปี" />
                      <v-textarea v-model="section.data.subheadline" label="Sub-headline" rows="2" />
                      <image-field v-model="section.data.image_url" label="Hero Image" :suggestions="productImages" @upload="uploadImage" />
                      <v-text-field v-model="section.data.cta_text" label="CTA Button Text" placeholder="สั่งซื้อตอนนี้" />
                    </template>

                    <!-- PAIN -->
                    <template v-else-if="section.type === 'pain'">
                      <v-text-field v-model="section.data.title" label="Title" placeholder="คุณเคยเจอแบบนี้ไหม?" />
                      <div v-for="(_, j) in section.data.items" :key="j" class="d-flex align-center">
                        <v-text-field v-model="section.data.items[j]" :label="`Point ${j + 1}`" class="mr-2" />
                        <v-btn icon="mdi-close" size="x-small" variant="text" color="error"
                          @click="section.data.items.splice(j, 1)" :disabled="section.data.items.length === 1" />
                      </div>
                      <v-btn size="small" variant="tonal" prepend-icon="mdi-plus" class="text-none"
                        @click="section.data.items.push('')">Add Point</v-btn>
                    </template>

                    <!-- STORY -->
                    <template v-else-if="section.type === 'story'">
                      <v-text-field v-model="section.data.title" label="Title" />
                      <v-textarea v-model="section.data.body" label="Story (blank line = new paragraph)" rows="5" />
                      <image-field v-model="section.data.image_url" label="Story Image" :suggestions="productImages" @upload="uploadImage" />
                    </template>

                    <!-- SHOWCASE -->
                    <template v-else-if="section.type === 'showcase'">
                      <v-text-field v-model="section.data.title" label="Title" />
                      <div v-for="(_, j) in section.data.images" :key="j" class="d-flex align-center">
                        <image-field v-model="section.data.images[j]" :label="`Image ${j + 1}`" :suggestions="productImages" class="flex-grow-1 mr-2" @upload="uploadImage" />
                        <v-btn icon="mdi-close" size="x-small" variant="text" color="error"
                          @click="section.data.images.splice(j, 1)" />
                      </div>
                      <v-btn size="small" variant="tonal" prepend-icon="mdi-plus" class="text-none mb-3"
                        @click="section.data.images.push('')">Add Image</v-btn>
                      <v-text-field v-model="section.data.caption" label="Caption (optional)" />
                    </template>

                    <!-- OFFER STACK -->
                    <template v-else-if="section.type === 'offer'">
                      <v-text-field v-model="section.data.title" label="Title" placeholder="สิ่งที่คุณจะได้รับ" />
                      <div v-for="(item, j) in section.data.items" :key="j" class="d-flex align-center ga-2">
                        <v-text-field v-model="item.name" :label="`Item ${j + 1}`" class="flex-grow-1" />
                        <v-text-field v-model.number="item.value" label="Value (THB)" type="number" style="max-width: 140px" />
                        <v-btn icon="mdi-close" size="x-small" variant="text" color="error"
                          @click="section.data.items.splice(j, 1)" :disabled="section.data.items.length === 1" />
                      </div>
                      <v-btn size="small" variant="tonal" prepend-icon="mdi-plus" class="text-none mb-3"
                        @click="section.data.items.push({ name: '', value: 0 })">Add Item</v-btn>
                      <v-text-field v-model="section.data.note" label="Note under the stack (optional)" />
                      <v-alert type="info" variant="tonal" density="compact" class="mt-2">
                        Total value is summed automatically and shown crossed out next to the offer price.
                      </v-alert>
                    </template>

                    <!-- TESTIMONIALS -->
                    <template v-else-if="section.type === 'testimonials'">
                      <v-text-field v-model="section.data.title" label="Title" placeholder="เสียงจากลูกค้า" />
                      <v-card v-for="(item, j) in section.data.items" :key="j" variant="outlined" rounded="lg" class="mb-2 pa-3">
                        <v-textarea v-model="item.quote" label="Quote" rows="2" hide-details class="mb-2" />
                        <div class="d-flex align-center">
                          <v-text-field v-model="item.name" label="Customer name" hide-details class="mr-2" />
                          <v-btn icon="mdi-close" size="x-small" variant="text" color="error"
                            @click="section.data.items.splice(j, 1)" :disabled="section.data.items.length === 1" />
                        </div>
                      </v-card>
                      <v-btn size="small" variant="tonal" prepend-icon="mdi-plus" class="text-none"
                        @click="section.data.items.push({ quote: '', name: '' })">Add Testimonial</v-btn>
                    </template>

                    <!-- GUARANTEE -->
                    <template v-else-if="section.type === 'guarantee'">
                      <v-text-field v-model="section.data.title" label="Title" placeholder="การรับประกันจากเรา" />
                      <v-textarea v-model="section.data.body" label="Guarantee text" rows="3"
                        placeholder="เปลี่ยนไซซ์ได้ภายใน 7 วัน..." />
                    </template>

                    <!-- FAQ -->
                    <template v-else-if="section.type === 'faq'">
                      <v-text-field v-model="section.data.title" label="Title" placeholder="คำถามที่พบบ่อย" />
                      <v-card v-for="(item, j) in section.data.items" :key="j" variant="outlined" rounded="lg" class="mb-2 pa-3">
                        <v-text-field v-model="item.q" label="Question" hide-details class="mb-2" />
                        <div class="d-flex align-start">
                          <v-textarea v-model="item.a" label="Answer" rows="2" hide-details class="mr-2" />
                          <v-btn icon="mdi-close" size="x-small" variant="text" color="error"
                            @click="section.data.items.splice(j, 1)" :disabled="section.data.items.length === 1" />
                        </div>
                      </v-card>
                      <v-btn size="small" variant="tonal" prepend-icon="mdi-plus" class="text-none"
                        @click="section.data.items.push({ q: '', a: '' })">Add FAQ</v-btn>
                    </template>
                  </v-expansion-panel-text>
                </v-expansion-panel>
              </v-expansion-panels>
            </v-window-item>
          </v-window>

          <v-alert v-if="editorError" type="error" variant="tonal" density="compact" class="mt-2">{{ editorError }}</v-alert>
        </v-card-text>

        <v-divider />
        <v-card-actions class="pa-5">
          <v-chip :color="form.status === 'published' ? 'success' : 'grey'" variant="tonal" label class="text-capitalize mr-2">
            {{ form.status }}
          </v-chip>
          <v-switch
            :model-value="form.status === 'published'"
            @update:model-value="form.status = $event ? 'published' : 'draft'"
            label="Published"
            color="success"
            hide-details
            class="flex-grow-0"
          />
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="editorDialog = false">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" class="text-none px-6" @click="savePage">
            {{ editingId ? 'Save' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirm -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card>
        <v-card-title class="pa-5 pb-2 text-h6 font-weight-bold">Delete Sale Page</v-card-title>
        <v-card-text class="px-5">
          Delete <strong>{{ deletingPage?.title }}</strong>? The link /s/{{ deletingPage?.slug }} will stop working.
          Orders placed through it are kept.
        </v-card-text>
        <v-card-actions class="pa-5 pt-2">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" class="text-none px-6" :loading="saving" @click="deletePage">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar" timeout="2000" color="success">{{ snackbarText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, defineComponent, h } from 'vue'
import { useDisplay } from 'vuetify'
import { VTextField } from 'vuetify/components/VTextField'
import { VBtn } from 'vuetify/components/VBtn'
import api from '@/services/api'

// Small inline component: URL text field + upload button + clickable
// thumbnails of the selected product's existing images ("suggestions"), so
// pages can be built from photos already in the system without re-uploading.
// Emits 'upload' with (file, setUrl) so the parent owns the API call.
const ImageField = defineComponent({
  name: 'ImageField',
  props: {
    modelValue: { type: String, default: '' },
    label: { type: String, default: 'Image' },
    suggestions: { type: Array as () => string[], default: () => [] },
  },
  emits: ['update:modelValue', 'upload'],
  setup(props, { emit }) {
    const uploading = ref(false)
    function pick() {
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = 'image/*'
      input.onchange = () => {
        const file = input.files?.[0]
        if (!file) return
        uploading.value = true
        emit('upload', file, (url: string) => {
          emit('update:modelValue', url)
          uploading.value = false
        })
      }
      input.click()
    }
    return () =>
      h('div', { class: 'mb-2' }, [
        h('div', { class: 'd-flex align-center' }, [
          h(VTextField, {
            modelValue: props.modelValue,
            label: props.label,
            placeholder: '/uploads/… or https://…',
            hideDetails: true,
            class: 'mr-2',
            'onUpdate:modelValue': (v: string) => emit('update:modelValue', v),
          }),
          h(VBtn, {
            class: 'text-none',
            variant: 'tonal',
            color: 'secondary',
            loading: uploading.value,
            onClick: pick,
          }, () => 'Upload'),
        ]),
        props.suggestions.length
          ? h('div', { class: 'd-flex align-center flex-wrap ga-2 mt-2' }, [
              h('span', { class: 'text-caption text-medium-emphasis' }, 'รูปจากสินค้า:'),
              ...props.suggestions.map((url) =>
                h('img', {
                  src: url,
                  style: {
                    width: '44px', height: '56px', objectFit: 'cover', cursor: 'pointer',
                    borderRadius: '6px',
                    border: props.modelValue === url ? '2px solid #C4A24D' : '1px solid rgba(0,0,0,.15)',
                  },
                  title: 'คลิกเพื่อใช้รูปนี้',
                  onClick: () => emit('update:modelValue', url),
                })
              ),
            ])
          : null,
      ])
  },
})
const imageField = ImageField

const { mobile } = useDisplay()

const headers = [
  { title: 'Page', key: 'title' },
  { title: 'Offer', key: 'offer', sortable: false },
  { title: 'Status', key: 'status' },
  { title: 'Performance', key: 'stats', sortable: false },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '210px' },
]

const sectionMeta: Record<string, { label: string; icon: string }> = {
  hero: { label: 'Hero — headline + CTA', icon: 'mdi-format-title' },
  pain: { label: 'Pain / Promise — bullet points', icon: 'mdi-help-circle-outline' },
  story: { label: 'Story — brand narrative', icon: 'mdi-book-open-page-variant-outline' },
  showcase: { label: 'Product Showcase — gallery', icon: 'mdi-image-multiple-outline' },
  offer: { label: 'Offer Stack — value build-up', icon: 'mdi-gift-open-outline' },
  testimonials: { label: 'Social Proof — testimonials', icon: 'mdi-format-quote-close' },
  guarantee: { label: 'Guarantee', icon: 'mdi-shield-check-outline' },
  faq: { label: 'FAQ', icon: 'mdi-frequently-asked-questions' },
}

const defaultSections = () => ([
  { type: 'hero', enabled: true, data: { kicker: 'Limited Release', headline: '', subheadline: '', image_url: '', cta_text: 'สั่งซื้อตอนนี้' } },
  { type: 'pain', enabled: false, data: { title: 'คุณเคยเจอแบบนี้ไหม?', items: ['', '', ''] } },
  { type: 'story', enabled: false, data: { title: '', body: '', image_url: '' } },
  { type: 'showcase', enabled: false, data: { title: '', images: [''], caption: '' } },
  { type: 'offer', enabled: true, data: { title: 'สิ่งที่คุณจะได้รับ', items: [{ name: '', value: 0 }], note: '' } },
  { type: 'testimonials', enabled: false, data: { title: 'เสียงจากลูกค้า', items: [{ quote: '', name: '' }] } },
  { type: 'guarantee', enabled: false, data: { title: 'การรับประกันจากเรา', body: '' } },
  { type: 'faq', enabled: false, data: { title: 'คำถามที่พบบ่อย', items: [{ q: '', a: '' }] } },
])

const pages = ref<any[]>([])
const products = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const editorDialog = ref(false)
const deleteDialog = ref(false)
const editorTab = ref('offer')
const editorError = ref('')
const editingId = ref<number | null>(null)
const deletingPage = ref<any>(null)
const snackbar = ref(false)
const snackbarText = ref('')

const emptyForm = () => ({
  title: '',
  slug: '',
  status: 'draft',
  product_id: 0,
  offer_price: null as number | null,
  sections: defaultSections() as any[],
  bump_enabled: false,
  bump_product_id: null as number | null,
  bump_price: 0,
  bump_headline: '',
  bump_description: '',
  countdown_ends_at: null as string | null,
  show_stock: false,
  allow_coupon: false,
})
const form = ref(emptyForm())

// offer_price is nullable in the API; the input maps 0/empty to null.
const offerPriceInput = computed({
  get: () => form.value.offer_price ?? 0,
  set: (v: number) => { form.value.offer_price = v > 0 ? v : null },
})

// datetime-local input <-> ISO string
const countdownInput = computed({
  get: () => {
    if (!form.value.countdown_ends_at) return ''
    const d = new Date(form.value.countdown_ends_at)
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
  },
  set: (v: string) => { form.value.countdown_ends_at = v ? new Date(v).toISOString() : null },
})

const selectedProduct = computed(() => products.value.find((p: any) => p.id === form.value.product_id))
const bumpProduct = computed(() => products.value.find((p: any) => p.id === form.value.bump_product_id))

// All images of the selected main product — offered as one-click suggestions
// on every image field in the section editor.
const productImages = computed<string[]>(() => {
  const p = selectedProduct.value
  if (!p) return []
  return [...new Set([p.image_url, ...(p.images || [])].filter(Boolean))]
})

// When a product is picked, pre-fill empty image slots from its gallery so the
// page has visuals immediately (the storefront falls back to product images
// anyway — this makes the same thing visible in the editor).
watch(() => form.value.product_id, () => {
  const imgs = productImages.value
  if (!imgs.length) return
  for (const section of form.value.sections) {
    if (section.type === 'hero' && !section.data.image_url) {
      section.data.image_url = imgs[0]
    }
    if (section.type === 'showcase') {
      const existing = (section.data.images || []).filter((x: string) => x && x.trim())
      if (!existing.length) section.data.images = [...imgs]
    }
  }
})

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB', maximumFractionDigits: 0 }).format(n)
}

function conversionRate(p: any) {
  if (!p.views) return '—'
  return `${((p.orders_count / p.views) * 100).toFixed(1)}%`
}

function bumpRate(p: any) {
  if (!p.orders_count) return '—'
  return `${((p.bump_count / p.orders_count) * 100).toFixed(0)}%`
}

async function fetchPages() {
  loading.value = true
  try {
    const { data } = await api.get('/sale-pages')
    pages.value = data || []
  } finally {
    loading.value = false
  }
}

async function fetchProducts() {
  const { data } = await api.get('/products')
  products.value = data || []
}

// Merge saved sections with defaults so pages created before a new block type
// existed still show every panel in the editor.
function mergeSections(saved: any[]): any[] {
  const defaults = defaultSections()
  const byType = new Map((saved || []).map((s: any) => [s.type, s]))
  const merged = (saved || []).filter((s: any) => sectionMeta[s.type])
  for (const d of defaults) {
    if (!byType.has(d.type)) merged.push(d)
  }
  return merged
}

function slugify(s: string) {
  return s.toLowerCase().trim().replace(/[^a-z0-9ก-๙\s-]/g, '').replace(/[\sก-๙]+/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '')
}

function openCreate() {
  editingId.value = null
  editorError.value = ''
  editorTab.value = 'offer'
  form.value = emptyForm()
  editorDialog.value = true
}

function openEdit(p: any) {
  editingId.value = p.id
  editorError.value = ''
  editorTab.value = 'offer'
  form.value = {
    title: p.title,
    slug: p.slug,
    status: p.status,
    product_id: p.product_id,
    offer_price: p.offer_price,
    sections: mergeSections(JSON.parse(JSON.stringify(p.sections || []))),
    bump_enabled: p.bump_enabled,
    bump_product_id: p.bump_product_id,
    bump_price: p.bump_price,
    bump_headline: p.bump_headline,
    bump_description: p.bump_description,
    countdown_ends_at: p.countdown_ends_at,
    show_stock: p.show_stock,
    allow_coupon: p.allow_coupon,
  }
  editorDialog.value = true
}

function moveSection(i: number, dir: number) {
  const target = i + dir
  const sections = form.value.sections
  if (target < 0 || target >= sections.length) return
  const [moved] = sections.splice(i, 1)
  sections.splice(target, 0, moved)
}

async function savePage() {
  editorError.value = ''
  if (!form.value.slug && form.value.title) {
    form.value.slug = slugify(form.value.title)
  }
  saving.value = true
  try {
    const payload = { ...form.value }
    if (editingId.value) {
      await api.put(`/sale-pages/${editingId.value}`, payload)
    } else {
      await api.post('/sale-pages', payload)
    }
    editorDialog.value = false
    await fetchPages()
  } catch (err: any) {
    editorError.value = err.response?.data?.error || 'Failed to save page'
  } finally {
    saving.value = false
  }
}

async function togglePublish(p: any) {
  await api.post(`/sale-pages/${p.id}/toggle`)
  await fetchPages()
}

async function duplicatePage(p: any) {
  await api.post(`/sale-pages/${p.id}/duplicate`)
  await fetchPages()
}

function confirmDelete(p: any) {
  deletingPage.value = p
  deleteDialog.value = true
}

async function deletePage() {
  saving.value = true
  try {
    await api.delete(`/sale-pages/${deletingPage.value.id}`)
    deleteDialog.value = false
    await fetchPages()
  } finally {
    saving.value = false
  }
}

function copyLink(p: any) {
  navigator.clipboard.writeText(`${window.location.origin}/s/${p.slug}`)
  snackbarText.value = 'Link copied'
  snackbar.value = true
}

// Shared upload used by every ImageField in the section editor.
async function uploadImage(file: File, setUrl: (url: string) => void) {
  try {
    const fd = new FormData()
    fd.append('image', file)
    const { data } = await api.post('/sale-pages/upload', fd)
    setUrl(data.url)
  } catch {
    setUrl('')
  }
}

onMounted(() => {
  fetchPages()
  fetchProducts()
})
</script>
