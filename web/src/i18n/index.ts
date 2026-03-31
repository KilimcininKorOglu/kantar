import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import tr from './locales/tr.json'
import de from './locales/de.json'

const LOCALE_KEY = 'kantar_locale'

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    tr: { translation: tr },
    de: { translation: de },
  },
  lng: localStorage.getItem(LOCALE_KEY) || 'en',
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
})

export default i18n

export function setLocale(lng: string) {
  i18n.changeLanguage(lng)
  localStorage.setItem(LOCALE_KEY, lng)
}

export function getLocale(): string {
  return i18n.language
}

export const supportedLocales = [
  { code: 'en', label: 'English' },
  { code: 'tr', label: 'Turkce' },
  { code: 'de', label: 'Deutsch' },
]
