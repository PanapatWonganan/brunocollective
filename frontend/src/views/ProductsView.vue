<template>
  <div>
    <div class="d-flex align-center mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Products</div>
        <div class="text-caption text-medium-emphasis">Manage your product inventory</div>
      </div>
      <v-spacer />
      <v-btn variant="tonal" color="secondary" prepend-icon="mdi-swap-vertical" @click="openReorder" class="text-none mr-2">
        จัดเรียงหน้าร้าน
      </v-btn>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openDialog()" class="text-none">
        Add Product
      </v-btn>
    </div>

    <!-- Stats Row -->
    <v-row class="mb-4">
      <v-col cols="12" sm="4">
        <v-card class="mini-stat" border="false" style="border-left: 3px solid #C4A24D !important;">
          <v-card-text class="d-flex align-center pa-4">
            <v-icon icon="mdi-package-variant" color="primary" class="mr-3" />
            <div>
              <div class="text-caption text-medium-emphasis">Total Products</div>
              <div class="text-h6 font-weight-bold">{{ products.length }}</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card class="mini-stat" border="false" style="border-left: 3px solid #5D7A5F !important;">
          <v-card-text class="d-flex align-center pa-4">
            <v-icon icon="mdi-check-circle" color="success" class="mr-3" />
            <div>
              <div class="text-caption text-medium-emphasis">In Stock</div>
              <div class="text-h6 font-weight-bold">{{ products.filter(p => totalStock(p) > 5).length }}</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card class="mini-stat" border="false" style="border-left: 3px solid #9B3B3B !important;">
          <v-card-text class="d-flex align-center pa-4">
            <v-icon icon="mdi-alert" color="error" class="mr-3" />
            <div>
              <div class="text-caption text-medium-emphasis">Low Stock</div>
              <div class="text-h6 font-weight-bold">{{ products.filter(p => totalStock(p) <= 5).length }}</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-card>
      <v-card-text class="pa-5">
        <v-text-field
          v-model="search"
          prepend-inner-icon="mdi-magnify"
          placeholder="Search products by name or SKU..."
          clearable
          hide-details
          @update:model-value="fetchProducts"
          class="mb-4"
          style="max-width: 400px;"
        />

        <v-data-table
          :headers="headers"
          :items="products"
          :loading="loading"
          items-per-page="10"
          class="rounded-lg"
        >
          <template v-slot:item.name="{ item }">
            <div class="d-flex align-center py-2">
              <v-avatar size="36" rounded="lg" color="primary" variant="tonal">
                <v-img v-if="item.image_url" :src="item.image_url" cover />
                <v-icon v-else icon="mdi-package-variant" size="18" />
              </v-avatar>
              <div class="ml-3">
                <div class="font-weight-medium">{{ item.name }}</div>
                <div class="text-caption text-medium-emphasis">
                  {{ item.sku || 'No SKU' }}<span v-if="item.category"> · {{ item.category }}</span>
                </div>
              </div>
            </div>
          </template>
          <template v-slot:item.variants="{ item }">
            <div v-if="item.variants && item.variants.length" class="d-flex flex-wrap ga-1 py-2">
              <v-chip v-for="(v, i) in item.variants" :key="i" variant="tonal" size="x-small" label color="secondary">
                {{ [v.size, v.color].filter(Boolean).join(' · ') || 'One size' }} · {{ v.stock }}
              </v-chip>
            </div>
            <span v-else class="text-medium-emphasis text-caption">
              {{ item.size ? `${item.size} (legacy)` : '— no variants' }}
            </span>
          </template>
          <template v-slot:item.price="{ item }">
            <span class="font-weight-medium">{{ formatCurrency(item.price) }}</span>
          </template>
          <template v-slot:item.total_stock="{ item }">
            <v-chip
              :color="totalStock(item) === 0 ? 'error' : totalStock(item) <= 5 ? 'warning' : 'success'"
              variant="tonal" size="small" label
            >
              <v-icon :icon="totalStock(item) === 0 ? 'mdi-close-circle' : totalStock(item) <= 5 ? 'mdi-alert-circle' : 'mdi-check-circle'" size="14" class="mr-1" />
              {{ totalStock(item) }}
            </v-chip>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-tooltip text="รวมเข้ากับสินค้าอื่น (สินค้าซ้ำ)" location="top">
              <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-merge" size="small" variant="text" color="secondary" @click="openMerge(item)" />
              </template>
            </v-tooltip>
            <v-btn icon="mdi-pencil-outline" size="small" variant="text" color="primary" @click="openDialog(item)" />
            <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" @click="confirmDelete(item)" />
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Create/Edit Dialog -->
    <v-dialog v-model="dialog" max-width="640" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">{{ editingProduct ? 'Edit Product' : 'New Product' }}</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-form ref="form">
            <v-text-field v-model="formData.name" label="Product Name" :rules="[v => !!v || 'Required']" class="mb-1" />
            <v-alert
              v-if="similarProduct"
              type="warning" variant="tonal" density="compact" icon="mdi-content-duplicate" class="mb-3"
            >
              <div class="text-body-2">
                มีสินค้าชื่อคล้ายกันอยู่แล้ว: <strong>{{ similarProduct.name }}</strong>
                ({{ similarProduct.sku || 'no SKU' }}) — ถ้าเป็นตัวเดียวกัน แก้ไขตัวเดิมแทนการสร้างใหม่
              </div>
            </v-alert>
            <v-text-field v-model="formData.sku" label="Base SKU" hint="optional — variants can have their own SKU" class="mb-1" />
            <v-textarea v-model="formData.description" label="Description" rows="2" class="mb-1" />
            <v-combobox
              v-model="formData.category"
              :items="categoryOptions"
              label="หมวดสินค้า (Category)"
              hint="ใช้จัดกลุ่มใน analytics — พิมพ์เพิ่มเองได้"
              persistent-hint
              class="mb-2"
            />
            <div class="d-flex ga-2">
              <v-text-field v-model.number="formData.price" label="Price (THB)" type="number" prefix="฿" :rules="[v => v >= 0 || 'Invalid']" class="mb-3" />
              <v-text-field
                v-model.number="formData.cost"
                label="ต้นทุน/ชิ้น (Cost)" type="number" prefix="฿"
                hint="ใช้คำนวณกำไร" persistent-hint
                :rules="[v => v >= 0 || 'Invalid']" class="mb-3"
              />
            </div>

            <!-- Variants (size + color + stock) -->
            <div class="d-flex align-center mb-2">
              <div class="text-subtitle-2 font-weight-medium">Variants (size / color)</div>
              <v-spacer />
              <v-btn size="small" variant="tonal" color="primary" prepend-icon="mdi-plus" class="text-none" @click="addVariant">
                Add variant
              </v-btn>
            </div>
            <div v-if="!formData.variants.length" class="mb-1">
              <div class="text-caption text-medium-emphasis mb-2">
                No variants yet. Add one per size/color combination — each holds its own stock.
                For products without sizes/colors, set the stock directly below.
              </div>
              <v-text-field
                v-model.number="formData.stock"
                label="Stock (สินค้าไม่มีไซส์/สี)"
                type="number"
                hint="จำนวนคงเหลือรวมของสินค้านี้ — ใช้เมื่อไม่มี variants"
                persistent-hint
                :rules="[v => v >= 0 || 'Invalid']"
                class="mb-3"
                style="max-width: 260px;"
              />
            </div>
            <div v-for="(v, i) in formData.variants" :key="i" class="variant-row mb-2">
              <v-combobox
                v-model="v.size" :items="sizeOptions" label="Size" density="compact"
                hide-details clearable class="variant-size"
              />
              <v-text-field v-model="v.color" label="Color" density="compact" hide-details class="variant-color" />
              <v-text-field v-model="v.sku" label="SKU" density="compact" hide-details class="variant-sku" />
              <v-text-field v-model.number="v.stock" label="Stock" type="number" density="compact" hide-details class="variant-stock" />
              <v-btn icon="mdi-close" size="small" variant="text" color="error" @click="removeVariant(i)" />
            </div>

            <!-- Product images -->
            <div class="text-subtitle-2 font-weight-medium mb-2 mt-2">Product Images</div>
            <div v-if="formData.images.length" class="d-flex flex-wrap ga-2 mb-3">
              <div v-for="img in formData.images" :key="img" class="img-thumb">
                <v-img :src="img" width="76" height="76" cover rounded="lg" />
                <v-btn
                  icon="mdi-close" size="x-small" color="error" variant="flat"
                  class="img-thumb-del" :loading="deletingImg === img"
                  @click="removeImage(img)"
                />
              </div>
            </div>
            <v-file-input
              v-model="pendingFiles"
              label="Add images"
              prepend-icon="mdi-camera"
              accept="image/*"
              multiple
              chips
              show-size
              density="comfortable"
              hint="Upload one or more photos. The first becomes the main image."
              persistent-hint
            />
          </v-form>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn @click="dialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveProduct" class="text-none px-6">
            {{ editingProduct ? 'Update' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Reorder storefront -->
    <v-dialog v-model="reorderDialog" max-width="560">
      <v-card>
        <v-card-title class="pa-5 pb-1">
          <span class="text-h6 font-weight-bold">จัดเรียงสินค้าหน้าร้าน</span>
        </v-card-title>
        <v-card-text class="px-5">
          <div class="text-caption text-medium-emphasis mb-3">
            ลากสลับตำแหน่ง หรือใช้ปุ่มลูกศร — อันดับ 1 แสดงเป็นตัวแรกทั้งหน้าแรกและหน้า The Collection
          </div>
          <div
            v-for="(p, i) in reorderList"
            :key="p.id"
            class="reorder-row"
            :class="{ 'reorder-dragging': dragIndex === i }"
            draggable="true"
            @dragstart="dragIndex = i"
            @dragover.prevent
            @drop="onReorderDrop(i)"
            @dragend="dragIndex = null"
          >
            <v-icon icon="mdi-drag-horizontal-variant" size="18" class="reorder-grip" />
            <span class="reorder-num">{{ i + 1 }}</span>
            <v-avatar size="34" rounded="lg" color="primary" variant="tonal">
              <v-img v-if="p.image_url" :src="p.image_url" cover />
              <v-icon v-else icon="mdi-package-variant" size="16" />
            </v-avatar>
            <div class="ml-3 flex-grow-1 overflow-hidden">
              <div class="font-weight-medium text-truncate">{{ p.name }}</div>
              <div class="text-caption text-medium-emphasis">{{ totalStock(p) }} in stock</div>
            </div>
            <v-btn icon="mdi-arrow-up" size="x-small" variant="text" :disabled="i === 0" @click="moveReorder(i, -1)" />
            <v-btn icon="mdi-arrow-down" size="x-small" variant="text" :disabled="i === reorderList.length - 1" @click="moveReorder(i, 1)" />
          </div>
        </v-card-text>
        <v-card-actions class="pa-5 pt-2">
          <v-spacer />
          <v-btn @click="reorderDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :loading="savingOrder" @click="saveReorder" class="text-none px-6">
            บันทึกลำดับ
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Merge Duplicate -->
    <v-dialog v-model="mergeDialog" max-width="520" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">รวมสินค้าซ้ำ</span>
        </v-card-title>
        <v-card-text class="px-5">
          <div class="text-body-2 mb-4">
            รวม "<strong>{{ mergingProduct?.name }}</strong>"
            <span class="text-medium-emphasis">({{ mergingProduct?.sku || 'no SKU' }})</span>
            เข้ากับสินค้าตัวหลักที่เลือกด้านล่าง
          </div>
          <v-autocomplete
            v-model="mergeTargetId"
            :items="mergeTargetOptions"
            label="สินค้าตัวหลัก (ตัวที่จะเก็บไว้)"
            placeholder="ค้นหาด้วยชื่อหรือ SKU"
            clearable
          />
          <v-alert type="warning" variant="tonal" density="comfortable" icon="mdi-merge" class="mt-2">
            <div class="text-body-2">
              ยอดขายและประวัติออเดอร์ทั้งหมดจะย้ายไปรวมที่ตัวหลัก
              สต็อกจะรวมกันตามไซส์/สี แล้ว "<strong>{{ mergingProduct?.name }}</strong>" จะถูกลบ —
              ราคาและรายละเอียดใช้ของตัวหลัก ย้อนกลับไม่ได้
            </div>
          </v-alert>
        </v-card-text>
        <v-card-actions class="pa-5 pt-3">
          <v-spacer />
          <v-btn @click="mergeDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="primary" :disabled="!mergeTargetId" :loading="merging" @click="doMerge" class="text-none px-6">
            รวมสินค้า
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="4000">
      {{ snackbar.text }}
    </v-snackbar>

    <!-- Delete Confirm -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card class="text-center pa-2">
        <v-card-text class="pt-5">
          <v-avatar color="error" variant="tonal" size="56" class="mb-4">
            <v-icon icon="mdi-delete-outline" size="28" />
          </v-avatar>
          <div class="text-h6 font-weight-bold mb-2">Delete Product?</div>
          <div class="text-body-2 text-medium-emphasis">
            Are you sure you want to delete "<strong>{{ deletingProduct?.name }}</strong>"? This action cannot be undone.
          </div>
        </v-card-text>
        <v-card-actions class="justify-center pb-5">
          <v-btn @click="deleteDialog = false" variant="text" class="text-none">Cancel</v-btn>
          <v-btn color="error" :loading="saving" @click="deleteProduct" class="text-none px-6">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import api from '@/services/api'

interface Variant {
  id?: number; product_id?: number; size: string; color: string; sku: string; stock: number;
}

interface Product {
  id?: number; name: string; sku: string; size: string; description: string;
  category: string; price: number; cost: number; stock: number;
  image_url: string; images: string[];
  variants: Variant[]; total_stock?: number;
}

const headers = [
  { title: 'Product', key: 'name' },
  { title: 'Variants', key: 'variants', sortable: false },
  { title: 'Price', key: 'price', align: 'end' as const },
  { title: 'Total Stock', key: 'total_stock', align: 'center' as const },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '100px' },
]

const sizeOptions = ['S', 'M', 'L', 'XL', 'XXL', 'Free Size', '38', '39', '40', '41', '42', '43', '44', '45']

const categoryOptions = ['เสื้อยืด', 'เสื้อเชิ้ต', 'เสื้อกันหนาว', 'กางเกง', 'กระโปรง', 'เดรส', 'รองเท้า', 'กระเป๋า', 'เครื่องประดับ', 'อื่นๆ']

// Total stock for a product: sum of variant stock, else the legacy stock field.
function totalStock(p: Product): number {
  if (p.variants && p.variants.length) return p.variants.reduce((n, v) => n + (Number(v.stock) || 0), 0)
  return Number(p.stock) || 0
}

// Short summary of variants for the list, e.g. "S·Black 5, M·White 3".
function variantSummary(p: Product): string {
  if (!p.variants || !p.variants.length) return p.size || '—'
  return p.variants
    .map(v => `${[v.size, v.color].filter(Boolean).join('·') || 'One size'} ${v.stock}`)
    .join(', ')
}

const products = ref<Product[]>([])
const search = ref('')
const loading = ref(false)
const dialog = ref(false)
const deleteDialog = ref(false)
const saving = ref(false)
const editingProduct = ref<Product | null>(null)
const deletingProduct = ref<Product | null>(null)
const form = ref()
const pendingFiles = ref<File[]>([])
const deletingImg = ref<string | null>(null)

const emptyForm = (): Product => ({ name: '', sku: '', size: '', description: '', category: '', price: 0, cost: 0, stock: 0, image_url: '', images: [], variants: [] })
const formData = ref<Product>(emptyForm())

function addVariant() {
  formData.value.variants.push({ size: '', color: '', sku: '', stock: 0 })
}
function removeVariant(index: number) {
  formData.value.variants.splice(index, 1)
}

function formatCurrency(n: number) {
  return new Intl.NumberFormat('th-TH', { style: 'currency', currency: 'THB' }).format(n)
}

async function fetchProducts() {
  loading.value = true
  const { data } = await api.get('/products', { params: { search: search.value } })
  products.value = data || []
  loading.value = false
}

function openDialog(product?: Product) {
  editingProduct.value = product || null
  // Clone, ensuring images/variants are always arrays (backend may send null).
  formData.value = product
    ? { ...product, images: [...(product.images || [])], variants: (product.variants || []).map(v => ({ ...v })) }
    : emptyForm()
  pendingFiles.value = []
  dialog.value = true
}

async function uploadPendingFiles(productId: number) {
  if (!pendingFiles.value.length) return
  const fd = new FormData()
  pendingFiles.value.forEach(f => fd.append('images', f))
  const { data } = await api.post(`/products/${productId}/images`, fd)
  // Reflect the new gallery back into the form so thumbnails update.
  formData.value.images = data.images || []
  pendingFiles.value = []
}

async function saveProduct() {
  saving.value = true
  try {
    let productId: number
    if (editingProduct.value) {
      await api.put(`/products/${editingProduct.value.id}`, formData.value)
      productId = editingProduct.value.id!
    } else {
      const { data } = await api.post('/products', formData.value)
      productId = data.id
      editingProduct.value = data
    }
    await uploadPendingFiles(productId)
    dialog.value = false
    await fetchProducts()
  } finally {
    saving.value = false
  }
}

async function removeImage(img: string) {
  if (!editingProduct.value?.id) {
    // Product not saved yet — nothing on the server to delete.
    formData.value.images = formData.value.images.filter(i => i !== img)
    return
  }
  deletingImg.value = img
  try {
    const { data } = await api.delete(`/products/${editingProduct.value.id}/images`, { data: { image: img } })
    formData.value.images = data.images || []
  } finally {
    deletingImg.value = null
  }
}

// Warn when a NEW product's name looks like an existing one — the path staff
// took to create duplicates in the first place (SKU uniqueness didn't catch it
// because they typed a different SKU).
const similarProduct = computed(() => {
  if (editingProduct.value) return null
  const name = formData.value.name.trim().toLowerCase()
  if (name.length < 3) return null
  return (
    products.value.find(p => {
      const other = p.name.trim().toLowerCase()
      return other === name || other.includes(name) || name.includes(other)
    }) || null
  )
})

// ---- Storefront reorder ----
// The admin /products list comes back in storefront order already, so the
// dialog starts from exactly what shoppers currently see.
const reorderDialog = ref(false)
const savingOrder = ref(false)
const reorderList = ref<Product[]>([])
const dragIndex = ref<number | null>(null)

function openReorder() {
  reorderList.value = [...products.value]
  dragIndex.value = null
  reorderDialog.value = true
}

function moveReorder(index: number, dir: number) {
  const target = index + dir
  if (target < 0 || target >= reorderList.value.length) return
  const list = reorderList.value
  ;[list[index], list[target]] = [list[target], list[index]]
}

function onReorderDrop(index: number) {
  if (dragIndex.value === null || dragIndex.value === index) return
  const list = reorderList.value
  const [moved] = list.splice(dragIndex.value, 1)
  list.splice(index, 0, moved)
  dragIndex.value = null
}

async function saveReorder() {
  savingOrder.value = true
  try {
    await api.put('/products/reorder', { ids: reorderList.value.map(p => p.id) })
    reorderDialog.value = false
    snackbar.text = 'บันทึกลำดับสินค้าแล้ว — หน้าร้านอัปเดตภายในไม่กี่วินาที'
    snackbar.color = 'success'
    snackbar.show = true
    await fetchProducts()
  } catch {
    snackbar.text = 'บันทึกลำดับไม่สำเร็จ กรุณาลองใหม่'
    snackbar.color = 'error'
    snackbar.show = true
  } finally {
    savingOrder.value = false
  }
}

// ---- Merge duplicates ----
const mergeDialog = ref(false)
const merging = ref(false)
const mergingProduct = ref<Product | null>(null)
const mergeTargetId = ref<number | null>(null)
const snackbar = reactive({ show: false, text: '', color: 'success' })

const mergeTargetOptions = computed(() =>
  products.value
    .filter(p => p.id !== mergingProduct.value?.id)
    .map(p => ({ title: `${p.name} (${p.sku || 'no SKU'})`, value: p.id! }))
)

function openMerge(product: Product) {
  mergingProduct.value = product
  mergeTargetId.value = null
  mergeDialog.value = true
}

async function doMerge() {
  if (!mergingProduct.value?.id || !mergeTargetId.value) return
  merging.value = true
  try {
    const { data } = await api.post(`/products/${mergingProduct.value.id}/merge`, {
      target_id: mergeTargetId.value,
    })
    mergeDialog.value = false
    snackbar.text = `รวมสินค้าเรียบร้อย — ย้ายประวัติออเดอร์ ${data.moved_items} รายการไปที่ "${data.product?.name}"`
    snackbar.color = 'success'
    snackbar.show = true
    await fetchProducts()
  } catch {
    snackbar.text = 'รวมสินค้าไม่สำเร็จ กรุณาลองใหม่'
    snackbar.color = 'error'
    snackbar.show = true
  } finally {
    merging.value = false
  }
}

function confirmDelete(product: Product) {
  deletingProduct.value = product
  deleteDialog.value = true
}

async function deleteProduct() {
  saving.value = true
  try {
    await api.delete(`/products/${deletingProduct.value?.id}`)
    deleteDialog.value = false
    await fetchProducts()
  } finally {
    saving.value = false
  }
}

onMounted(fetchProducts)
</script>

<style scoped>
.mini-stat {
  border: 1px solid #E8E2D9 !important;
}
.img-thumb {
  position: relative;
}
.img-thumb-del {
  position: absolute;
  top: -8px;
  right: -8px;
}
.variant-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.reorder-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid #E5E7EB;
  border-radius: 10px;
  margin-bottom: 8px;
  background: #fff;
  cursor: grab;
}
.reorder-dragging {
  opacity: 0.5;
  border-style: dashed;
}
.reorder-grip {
  color: #9CA3AF;
}
.reorder-num {
  min-width: 22px;
  text-align: center;
  font-weight: 600;
  color: #6B7280;
  font-size: 12px;
}
.variant-size { max-width: 130px; }
.variant-color { max-width: 130px; }
.variant-sku { flex: 1; min-width: 90px; }
.variant-stock { max-width: 90px; }
@media (max-width: 600px) {
  .variant-row { flex-wrap: wrap; }
  .variant-size, .variant-color, .variant-sku, .variant-stock { max-width: none; flex: 1 1 40%; }
}
</style>
