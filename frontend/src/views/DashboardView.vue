<template>
  <div>
    <v-tabs v-model="tab" color="secondary" class="mb-4 dashboard-tabs">
      <v-tab value="sales" prepend-icon="mdi-chart-line" class="text-none">ภาพรวมยอดขาย</v-tab>
      <v-tab value="inventory" prepend-icon="mdi-package-variant-closed" class="text-none">สต็อก</v-tab>
      <v-tab value="customers" prepend-icon="mdi-account-group-outline" class="text-none">ลูกค้า</v-tab>
      <v-tab value="products" prepend-icon="mdi-hanger" class="text-none">สินค้า</v-tab>
    </v-tabs>

    <!-- v-window mounts each tab lazily on first visit and keeps it alive after. -->
    <v-window v-model="tab">
      <v-window-item value="sales">
        <SalesTab />
      </v-window-item>
      <v-window-item value="inventory">
        <InventoryTab />
      </v-window-item>
      <v-window-item value="customers">
        <CustomersTab />
      </v-window-item>
      <v-window-item value="products">
        <ProductsTab />
      </v-window-item>
    </v-window>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SalesTab from './dashboard/SalesTab.vue'
import InventoryTab from './dashboard/InventoryTab.vue'
import CustomersTab from './dashboard/CustomersTab.vue'
import ProductsTab from './dashboard/ProductsTab.vue'

const tab = ref('sales')
</script>

<style scoped>
.dashboard-tabs {
  border-bottom: 1px solid #E5E7EB;
}
</style>
