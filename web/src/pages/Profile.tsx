import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { getTimezone, setTimezone as setTz, getTimezoneList } from '../utils/date'
import { setLocale, getLocale, supportedLocales } from '../i18n'
import { Globe, Languages, Mail, Lock, Check } from 'lucide-react'

export default function Profile() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [email, setEmail] = useState(user?.email || '')
  const [timezone, setTimezone] = useState(getTimezone())
  const [locale, setLang] = useState(getLocale())
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  // Password
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [pwSaving, setPwSaving] = useState(false)
  const [pwMessage, setPwMessage] = useState('')
  const [pwError, setPwError] = useState('')

  const handleSaveProfile = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true); setError(''); setSaved(false)
    try {
      await api.put('/profile', { email, timezone, locale })
      setTz(timezone)
      setLocale(locale)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError(t('profile.failedToUpdate'))
    }
    setSaving(false)
  }

  const handleChangePassword = async (e: FormEvent) => {
    e.preventDefault()
    setPwError(''); setPwMessage('')
    if (newPassword !== confirmPassword) {
      setPwError(t('profile.passwordMismatch'))
      return
    }
    setPwSaving(true)
    try {
      await api.put('/profile/password', { currentPassword, newPassword })
      setPwMessage(t('profile.passwordChanged'))
      setCurrentPassword(''); setNewPassword(''); setConfirmPassword('')
      setTimeout(() => setPwMessage(''), 3000)
    } catch {
      setPwError(t('profile.failedToChangePassword'))
    }
    setPwSaving(false)
  }

  return (
    <div className="space-y-5 max-w-2xl">
      {/* Profile Info */}
      <div className="bg-surface border border-border rounded p-4">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 bg-accent/20 rounded-full flex items-center justify-center">
            <span className="text-accent font-bold text-sm">{user?.username?.charAt(0).toUpperCase()}</span>
          </div>
          <div>
            <div className="text-sm font-semibold text-text">{user?.username}</div>
            <div className="text-[11px] text-text-dim font-mono">{user?.role}</div>
          </div>
        </div>
      </div>

      {/* Preferences */}
      <form onSubmit={handleSaveProfile} className="bg-surface border border-border rounded p-4">
        <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-4">{t('profile.preferences')}</h3>

        {error && <div className="bg-danger/10 border border-danger/20 text-danger text-xs rounded px-3 py-2 mb-3">{error}</div>}
        {saved && <div className="bg-success/10 border border-success/20 text-success text-xs rounded px-3 py-2 mb-3 flex items-center gap-1.5"><Check className="w-3 h-3" />{t('profile.profileUpdated')}</div>}

        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Mail className="w-3.5 h-3.5 text-text-dim" />
              <div>
                <div className="text-xs text-text">{t('common.email')}</div>
              </div>
            </div>
            <input value={email} onChange={e => setEmail(e.target.value)} type="email"
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56 focus:outline-none focus:border-accent font-mono" />
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Globe className="w-3.5 h-3.5 text-text-dim" />
              <div>
                <div className="text-xs text-text">{t('settings.timezone')}</div>
                <div className="text-[10px] text-text-dim">{t('settings.timezoneDesc')}</div>
              </div>
            </div>
            <select value={timezone} onChange={e => setTimezone(e.target.value)}
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56">
              {getTimezoneList().map(tz => <option key={tz} value={tz}>{tz}</option>)}
            </select>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Languages className="w-3.5 h-3.5 text-text-dim" />
              <div>
                <div className="text-xs text-text">{t('settings.language')}</div>
                <div className="text-[10px] text-text-dim">{t('settings.languageDesc')}</div>
              </div>
            </div>
            <select value={locale} onChange={e => setLang(e.target.value)}
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56">
              {supportedLocales.map(l => <option key={l.code} value={l.code}>{l.label}</option>)}
            </select>
          </div>
        </div>

        <div className="mt-4 flex justify-end">
          <button type="submit" disabled={saving}
            className="px-4 py-1.5 bg-accent hover:bg-accent-hover disabled:opacity-50 text-white text-xs font-medium rounded cursor-pointer">
            {saving ? t('common.saving') : t('common.save')}
          </button>
        </div>
      </form>

      {/* Change Password */}
      <form onSubmit={handleChangePassword} className="bg-surface border border-border rounded p-4">
        <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-4">{t('profile.changePassword')}</h3>

        {pwError && <div className="bg-danger/10 border border-danger/20 text-danger text-xs rounded px-3 py-2 mb-3">{pwError}</div>}
        {pwMessage && <div className="bg-success/10 border border-success/20 text-success text-xs rounded px-3 py-2 mb-3 flex items-center gap-1.5"><Check className="w-3 h-3" />{pwMessage}</div>}

        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Lock className="w-3.5 h-3.5 text-text-dim" />
              <span className="text-xs text-text">{t('profile.currentPassword')}</span>
            </div>
            <input type="password" value={currentPassword} onChange={e => setCurrentPassword(e.target.value)}
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56 focus:outline-none focus:border-accent" required />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Lock className="w-3.5 h-3.5 text-text-dim" />
              <span className="text-xs text-text">{t('profile.newPassword')}</span>
            </div>
            <input type="password" value={newPassword} onChange={e => setNewPassword(e.target.value)}
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56 focus:outline-none focus:border-accent" required minLength={8} />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Lock className="w-3.5 h-3.5 text-text-dim" />
              <span className="text-xs text-text">{t('profile.confirmPassword')}</span>
            </div>
            <input type="password" value={confirmPassword} onChange={e => setConfirmPassword(e.target.value)}
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56 focus:outline-none focus:border-accent" required minLength={8} />
          </div>
        </div>

        <div className="mt-4 flex justify-end">
          <button type="submit" disabled={pwSaving}
            className="px-4 py-1.5 bg-accent hover:bg-accent-hover disabled:opacity-50 text-white text-xs font-medium rounded cursor-pointer">
            {pwSaving ? t('common.saving') : t('profile.changePassword')}
          </button>
        </div>
      </form>
    </div>
  )
}
