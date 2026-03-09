import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

export default createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: 'light',
    themes: {
      light: {
        colors: {
          primary: '#1A1714',
          'primary-darken-1': '#0F0E0C',
          secondary: '#C4A24D',
          accent: '#C4A24D',
          success: '#5D7A5F',
          warning: '#D4A843',
          error: '#9B3B3B',
          info: '#6B7B8D',
          background: '#F7F3EE',
          surface: '#FFFFFF',
          'on-background': '#1A1714',
          'on-surface': '#3D3830',
        }
      }
    }
  },
  defaults: {
    VCard: { elevation: 0, rounded: 'xl', border: true },
    VBtn: { variant: 'elevated', rounded: 'lg' },
    VTextField: { variant: 'outlined', density: 'comfortable', rounded: 'lg', color: 'primary' },
    VSelect: { variant: 'outlined', density: 'comfortable', rounded: 'lg', color: 'primary' },
    VTextarea: { variant: 'outlined', density: 'comfortable', rounded: 'lg', color: 'primary' },
    VFileInput: { variant: 'outlined', density: 'comfortable', rounded: 'lg', color: 'primary' },
    VChip: { rounded: 'lg' },
    VDataTable: { hover: true },
  }
})
