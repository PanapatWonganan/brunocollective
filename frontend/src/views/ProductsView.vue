<template>
  <div>
    <div class="d-flex align-center mb-6">
      <div>
        <div class="text-h5 font-weight-bold">Products</div>
        <div class="text-caption text-medium-emphasis">Manage your product inventory</div>
      </div>
      <v-spacer />
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
              <div class="text-h6 font-weight-bold">{{ products.filter(p => p.stock > 5).length }}</div>
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
              <div class="text-h6 font-weight-bold">{{ products.filter(p => p.stock <= 5).length }}</div>
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
                <v-icon icon="mdi-package-variant" size="18" />
              </v-avatar>
              <div class="ml-3">
                <div class="font-weight-medium">{{ item.name }}</div>
                <div class="text-caption text-medium-emphasis">{{ item.sku || 'No SKU' }}</div>
              </div>
            </div>
          </template>
          <template v-slot:item.price="{ item }">
            <span class="font-weight-medium">{{ formatCurrency(item.price) }}</span>
          </template>
          <template v-slot:item.stock="{ item }">
            <v-chip
              :color="item.stock === 0 ? 'error' : item.stock <= 5 ? 'warning' : 'success'"
              variant="tonal" size="small" label
            >
              <v-icon :icon="item.stock === 0 ? 'mdi-close-circle' : item.stock <= 5 ? 'mdi-alert-circle' : 'mdi-check-circle'" size="14" class="mr-1" />
              {{ item.stock }}
            </v-chip>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-btn icon="mdi-pencil-outline" size="small" variant="text" color="primary" @click="openDialog(item)" />
            <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" @click="confirmDelete(item)" />
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Create/Edit Dialog -->
    <v-dialog v-model="dialog" max-width="520" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">{{ editingProduct ? 'Edit Product' : 'New Product' }}</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-form ref="form">
            <v-text-field v-model="formData.name" label="Product Name" :rules="[v => !!v || 'Required']" class="mb-1" />
            <v-text-field v-model="formData.sku" label="SKU" class="mb-1" />
            <v-textarea v-model="formData.description" label="Description" rows="2" class="mb-1" />
            <v-row>
              <v-col cols="6">
                <v-text-field v-model.number="formData.price" label="Price (THB)" type="number" prefix="฿" :rules="[v => v >= 0 || 'Invalid']" />
              </v-col>
              <v-col cols="6">
                <v-text-field v-model.number="formData.stock" label="Stock Quantity" type="number" :rules="[v => v >= 0 || 'Invalid']" />
              </v-col>
            </v-row>
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
import { ref, onMounted } from 'vue'
import api from '@/services/api'

interface Product {
  id?: number; name: string; sku: string; description: string;
  price: number; stock: number; image_url: string;
}

const headers = [
  { title: 'Product', key: 'name' },
  { title: 'Price', key: 'price', align: 'end' as const },
  { title: 'Stock', key: 'stock', align: 'center' as const },
  { title: '', key: 'actions', sortable: false, align: 'end' as const, width: '100px' },
]

const products = ref<Product[]>([])
const search = ref('')
const loading = ref(false)
const dialog = ref(false)
const deleteDialog = ref(false)
const saving = ref(false)
const editingProduct = ref<Product | null>(null)
const deletingProduct = ref<Product | null>(null)
const form = ref()

const emptyForm = (): Product => ({ name: '', sku: '', description: '', price: 0, stock: 0, image_url: '' })
const formData = ref<Product>(emptyForm())

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
  formData.value = product ? { ...product } : emptyForm()
  dialog.value = true
}

async function saveProduct() {
  saving.value = true
  try {
    if (editingProduct.value) {
      await api.put(`/products/${editingProduct.value.id}`, formData.value)
    } else {
      await api.post('/products', formData.value)
    }
    dialog.value = false
    await fetchProducts()
  } finally {
    saving.value = false
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
</style>
