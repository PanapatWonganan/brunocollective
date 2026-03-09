import { createApp } from 'vue'
import { createPinia } from 'pinia'
import vuetify from './plugins/vuetify'
import router from './router'
import VueApexCharts from 'vue3-apexcharts'
import App from './App.vue'

const app = createApp(App)
app.use(createPinia())
app.use(vuetify)
app.use(router)
app.use(VueApexCharts)
app.mount('#app')
