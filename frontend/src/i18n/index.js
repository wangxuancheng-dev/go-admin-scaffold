import { createI18n } from 'vue-i18n'
import en from './en.js'
import zh from './zh.js'

// Create i18n instance
const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: localStorage.getItem('language') || 'en',
  fallbackLocale: 'en',
  messages: {
    en,
    zh
  }
})

export default i18n 