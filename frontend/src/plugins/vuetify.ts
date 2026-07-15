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
          // Brand stays on primary/secondary (sidebar, buttons, active tabs).
          // Neutrals + status colors are high-contrast on white for readability.
          primary: '#1A1714',
          'primary-darken-1': '#0F0E0C',
          secondary: '#C4A24D',
          accent: '#C4A24D',
          success: '#15803D',
          warning: '#B45309',
          error: '#DC2626',
          info: '#2A78D6',
          background: '#F6F7F9',
          surface: '#FFFFFF',
          'on-background': '#111827',
          'on-surface': '#1F2937',
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
