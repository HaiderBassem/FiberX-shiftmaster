import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import type { ServiceCategory, ServicePlan } from './types';
import { useTranslation } from 'react-i18next';
import {
  ArrowRight, Plus, Pencil, Trash2, Wifi, Loader2, X,
  Clock, DollarSign, Zap, Server, Router, Cpu, StickyNote, Search,
} from 'lucide-react';

/* ─── Plan Detail Modal ────────────────────────────────── */
function PlanDetailModal({ plan, onClose }: { plan: ServicePlan; onClose: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
      <div className="bg-card border border-border rounded-2xl w-full max-w-2xl shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-card flex items-center justify-between px-6 py-5 border-b border-border z-10">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-primary/15 flex items-center justify-center">
              <Wifi className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h2 className="font-bold text-foreground">{plan.name}</h2>
              <p className="text-xs text-muted-foreground">{plan.category_name}</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>

        <div className="p-6 space-y-6">
          {/* Key metrics */}
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {[
              { icon: DollarSign, label: t('services.monthly_price'), value: `${plan.price.toLocaleString()} ${t('services.iqd_per_plan').split('/')[0]}`, color: 'text-emerald-400' },
              { icon: Clock, label: t('services.plan_duration'), value: `${plan.duration_days} ${t('services.days')}`, color: 'text-blue-400' },
              { icon: Zap, label: t('services.speed_label'), value: plan.speed ?? '—', color: 'text-primary' },
            ].map(item => (
              <div key={item.label} className="bg-background rounded-xl p-3 border border-border">
                <item.icon className={`w-4 h-4 mb-1.5 ${item.color}`} />
                <p className="text-muted-foreground text-xs">{item.label}</p>
                <p className="text-foreground font-semibold text-sm mt-0.5">{item.value}</p>
              </div>
            ))}
          </div>

          {/* Extra info */}
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Cpu className="w-4 h-4 shrink-0" />
              <span>{t('services.connection_type')} <span className="text-foreground font-medium">{plan.connection_type}</span></span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Server className="w-4 h-4 shrink-0" />
              <span>{t('services.max_data')} <span className="text-foreground font-medium">{plan.data_cap ?? t('services.unlimited')}</span></span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <DollarSign className="w-4 h-4 shrink-0" />
              <span>{t('services.install_fee_label')} <span className="text-foreground font-medium">{plan.installation_fee > 0 ? `${plan.installation_fee.toLocaleString()} ${t('services.iqd_per_plan').split('/')[0]}` : t('services.free')}</span></span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Router className="w-4 h-4 shrink-0" />
              <span>{t('services.router_label')} <span className="text-foreground font-medium">{plan.router_included ? t('services.yes') : t('services.no')}</span></span>
            </div>
          </div>

          {/* Description */}
          {plan.description && (
            <div className="bg-background rounded-xl p-4 border border-border">
              <p className="text-xs text-muted-foreground mb-1">{t('services.additional_details')}</p>
              <p className="text-foreground text-sm whitespace-pre-line">{plan.description}</p>
            </div>
          )}

          {/* Cabinet Notes */}
          {plan.cabinet_notes && (
            <div className="bg-amber-500/5 rounded-xl p-4 border border-amber-500/20">
              <div className="flex items-center gap-2 mb-2">
                <StickyNote className="w-4 h-4 text-amber-400" />
                <p className="text-amber-400 text-xs font-semibold">{t('services.cabinet_notes')}</p>
              </div>
              <p className="text-foreground text-sm whitespace-pre-line leading-relaxed">{plan.cabinet_notes}</p>
            </div>
          )}

          <div className="flex items-center justify-between text-xs text-muted-foreground pt-2 border-t border-border">
            <span>{t('services.created_by', { name: plan.creator_name ?? '—' })}</span>
            <span className={`px-2 py-0.5 rounded-full font-medium ${plan.is_active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-destructive/10 text-destructive'}`}>
              {plan.is_active ? t('services.active') : t('services.disabled')}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ─── Plan Form Modal ──────────────────────────────────── */
function PlanModal({ categoryId, initial, onClose, onSaved }: {
  categoryId: string; initial?: ServicePlan | null; onClose: () => void; onSaved: () => void;
}) {
  const { t } = useTranslation();
  const [f, setF] = useState({
    name: initial?.name ?? '',
    price: initial?.price?.toString() ?? '',
    duration_days: initial?.duration_days?.toString() ?? '30',
    speed: initial?.speed ?? '',
    data_cap: initial?.data_cap ?? 'Unlimited',
    connection_type: initial?.connection_type ?? 'ONU',
    installation_fee: initial?.installation_fee?.toString() ?? '0',
    router_included: initial?.router_included ?? false,
    description: initial?.description ?? '',
    cabinet_notes: initial?.cabinet_notes ?? '',
  });

  const isCustomDevice = !['ONU', 'ONT'].includes(f.connection_type);
  const [deviceSelect, setDeviceSelect] = useState(isCustomDevice && f.connection_type ? 'Other' : (f.connection_type || 'ONU'));
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');


  const set = (k: string, v: any) => setF(p => ({ ...p, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!f.name.trim() || !f.price) return setErr(t('services.req_fields'));
    setBusy(true); setErr('');
    try {
      const payload = {
        ...f,
        price: parseFloat(f.price),
        duration_days: parseInt(f.duration_days),
        installation_fee: parseFloat(f.installation_fee),
        speed: f.speed || undefined,
        description: f.description || undefined,
        cabinet_notes: f.cabinet_notes || undefined,
      };
      if (initial) {
        await api.put(`/services/plans/${initial.id}`, payload);
      } else {
        await api.post(`/services/categories/${categoryId}/plans`, payload);
      }
      onSaved(); onClose();
    } catch (e: any) {
      setErr(e.response?.data?.error ?? t('services.error_occurred'));
    } finally { setBusy(false); }
  };

  const field = (label: string, key: string, type = 'text', placeholder = '') => (
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
      <input type={type} value={(f as any)[key]} onChange={e => set(key, e.target.value)}
        placeholder={placeholder}
        className="w-full bg-background border border-border rounded-xl px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60" />
    </div>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
      <div className="bg-card border border-border rounded-2xl w-full max-w-2xl shadow-2xl max-h-[92vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-border shrink-0">
          <h2 className="font-bold text-foreground">{initial ? t('services.edit_plan') : t('services.new_plan')}</h2>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>
        <form onSubmit={submit} className="p-6 overflow-y-auto space-y-4">
          <div className="grid grid-cols-2 gap-4">
            {field(t('services.plan_name'), 'name', 'text', t('services.plan_name_ph'))}
            {field(t('services.price'), 'price', 'number', t('services.price_ph'))}
            {field(t('services.duration'), 'duration_days', 'number', t('services.duration_ph'))}
            {field(t('services.speed_label'), 'speed', 'text', t('services.speed_down_ph'))}
            {field(t('services.data_cap'), 'data_cap', 'text', t('services.data_cap_ph'))}
            {field(t('services.install_fee'), 'installation_fee', 'number', t('services.install_fee_ph'))}
          </div>


          <div className="grid grid-cols-2 gap-4">
            {/* Device Type */}
            <div className="col-span-2 sm:col-span-1">
              <label className="block text-xs font-medium text-muted-foreground mb-1">{t('services.device_type_label') || 'Device Type'}</label>
              <div className="flex gap-2">
                <select
                  value={deviceSelect}
                  onChange={e => {
                    setDeviceSelect(e.target.value);
                    if (e.target.value !== 'Other') {
                      set('connection_type', e.target.value);
                    } else {
                      set('connection_type', ''); // Clear for custom input
                    }
                  }}
                  className="bg-background border border-border rounded-xl px-3 py-2 text-sm text-foreground focus:outline-none focus:border-primary/60 flex-1"
                >
                  <option value="ONU">ONU</option>
                  <option value="ONT">ONT</option>
                  <option value="Other">Other</option>
                </select>
                {deviceSelect === 'Other' && (
                  <input
                    type="text"
                    value={f.connection_type}
                    onChange={e => set('connection_type', e.target.value)}
                    placeholder="Custom Device"
                    className="bg-background border border-border rounded-xl px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60 flex-1"
                  />
                )}
              </div>
            </div>

            {/* Router included */}
            <div className="flex items-center gap-3 pt-5 col-span-2 sm:col-span-1">
              <input type="checkbox" id="router" checked={f.router_included}
                onChange={e => set('router_included', e.target.checked)}
                className="w-4 h-4 accent-primary" />
              <label htmlFor="router" className="text-sm text-foreground">{t('services.router_included')}</label>
            </div>
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">{t('services.extra_desc')}</label>
            <textarea value={f.description} onChange={e => set('description', e.target.value)}
              rows={2} placeholder={t('services.extra_desc_ph')}
              className="w-full bg-background border border-border rounded-xl px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60 resize-none" />
          </div>

          {/* Cabinet Notes */}
          <div>
            <label className="block text-xs font-medium text-amber-400 mb-1 flex items-center gap-1.5">
              <StickyNote className="w-3.5 h-3.5" /> {t('services.cabinet_notes')}
            </label>
            <textarea value={f.cabinet_notes} onChange={e => set('cabinet_notes', e.target.value)}
              rows={3} placeholder={t('services.cabinet_notes_ph')}
              className="w-full bg-amber-500/5 border border-amber-500/20 rounded-xl px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-amber-500/40 resize-none" />
          </div>

          {err && <p className="text-destructive text-sm">{err}</p>}

          <div className="flex gap-3 pt-2 sticky bottom-0 bg-card pb-1">
            <button type="button" onClick={onClose}
              className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 text-sm font-medium">
              {t('services.cancel')}
            </button>
            <button type="submit" disabled={busy}
              className="flex-1 py-2.5 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium flex items-center justify-center gap-2">
              {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
              {initial ? t('services.save_changes') : t('services.create')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

/* ─── ServicePlans Page ────────────────────────────────── */
export function ServicePlans({ category, manager, onBack }: {
  category: ServiceCategory; manager: boolean; onBack: () => void;
}) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const key = ['service-plans', category.id];

  const [detailPlan, setDetailPlan] = useState<ServicePlan | null>(null);
  const [editPlan, setEditPlan] = useState<ServicePlan | null | undefined>(undefined);
  const [delPlan, setDelPlan] = useState<ServicePlan | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const { data: plans, isLoading } = useQuery<ServicePlan[]>({
    queryKey: key,
    queryFn: async () => (await api.get(`/services/categories/${category.id}/plans`)).data.data ?? [],
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/services/plans/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: key }); setDelPlan(null); },
  });

  const filteredPlans = (plans ?? []).filter(plan => 
    plan.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex-1">
          <h1 className="text-xl font-bold text-foreground flex items-center gap-2">
            <Wifi className="w-5 h-5 text-primary" /> {category.name}
          </h1>
          {category.description && (
            <p className="text-muted-foreground text-sm">{category.description}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {manager && (
            <button onClick={() => setEditPlan(null)}
              className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-all shadow-lg shadow-primary/20">
              <Plus className="w-4 h-4" /> {t('services.new_plan')}
            </button>
          )}
          <button onClick={onBack}
            className="p-2.5 rounded-xl border border-border bg-card hover:bg-muted transition-colors"
            title="Back">
            <ArrowRight className="w-5 h-5 text-muted-foreground rtl:-scale-x-100" />
          </button>
        </div>
      </div>

      {/* Search Bar */}
      <div className="relative max-w-md w-full">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t('services.search_plans_placeholder')}
          className="w-full pl-9 pr-4 py-2 bg-card border border-border rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-primary/20"
        />
      </div>

      {/* Plans grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="w-8 h-8 text-primary animate-spin" />
        </div>
      ) : !filteredPlans.length ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="w-16 h-16 rounded-2xl bg-muted/30 flex items-center justify-center mb-4">
            <Wifi className="w-8 h-8 text-muted-foreground" />
          </div>
          <p className="font-semibold text-foreground">{t('services.no_plans')}</p>
          <p className="text-muted-foreground text-sm mt-1">
            {manager ? t('services.no_plans_desc_manager') : t('services.no_plans_desc_employee')}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {filteredPlans.map(plan => (
            <div key={plan.id}
              className={`group relative border rounded-2xl p-5 hover:shadow-lg hover:shadow-primary/5 transition-all duration-200 cursor-pointer flex flex-col ${
                plan.is_active 
                  ? 'bg-card border-border hover:border-primary/40' 
                  : 'bg-muted/10 border-border/50 opacity-60 hover:opacity-100 grayscale-[30%] hover:grayscale-0'
              }`}
              onClick={() => setDetailPlan(plan)}>

              {/* Badges Row */}
              <div className="flex items-center justify-between mb-4">
                <div className="flex gap-1.5">
                  {plan.price === 0 ? (
                    <span className="text-xs px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-400 font-semibold">
                      {t('services.free')}
                    </span>
                  ) : (
                    <span className="text-xs px-2.5 py-1 rounded-full bg-blue-500/10 text-blue-400 font-semibold">
                      {t('services.paid')}
                    </span>
                  )}
                  {!plan.is_active && (
                    <div className="flex flex-col items-end gap-1">
                      <span className="text-xs px-2.5 py-1 rounded-full bg-destructive/10 text-destructive font-medium">
                        {t('services.disabled')}
                      </span>
                      {plan.disabled_at && (
                        <span className="text-[10px] text-muted-foreground/70">
                          Stopped: {new Date(plan.disabled_at).toLocaleDateString()}
                        </span>
                      )}
                    </div>
                  )}
                </div>
              </div>

              {/* Name */}
              <h3 className="font-bold text-foreground text-base mb-3 leading-snug">{plan.name}</h3>

              {/* Stats row */}
              <div className="grid grid-cols-2 gap-2 mb-4">
                <div className="bg-background rounded-xl p-2.5 text-center flex flex-col justify-center min-h-[60px]">
                  <p className="text-emerald-400 font-bold text-sm">
                    {plan.price === 0 ? t('services.free') : `${plan.price.toLocaleString()} د.ع`}
                  </p>
                  <p className="text-muted-foreground text-xs mt-0.5">{t('services.price_label')}</p>
                </div>
                <div className="bg-background rounded-xl p-2.5 text-center flex flex-col justify-center min-h-[60px]">
                  <p className="text-blue-400 font-bold text-sm">{plan.duration_days}</p>
                  <p className="text-muted-foreground text-xs mt-0.5">{t('services.days')}</p>
                </div>
              </div>

              {/* Speed & Connection details */}
              <div className="space-y-2.5 mt-auto pt-3 border-t border-border/50">
                {/* Connection/Device Type */}
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <Router className="w-3.5 h-3.5 text-primary" />
                    {t('services.device_type_label')}
                  </span>
                  <span className="text-foreground font-semibold">{plan.connection_type || 'FTTH'}</span>
                </div>

                {/* Speed (Single Speed) */}
                {plan.speed && (
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span className="flex items-center gap-1.5 text-muted-foreground">
                      <Zap className="w-3.5 h-3.5 text-primary" />
                      {t('services.speed_label')}
                    </span>
                    <span className="text-foreground font-semibold">{plan.speed}</span>
                  </div>
                )}

                {/* Router Info */}
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <Server className="w-3.5 h-3.5 text-primary" />
                    {t('services.router_label')}
                  </span>
                  <span className="text-foreground font-semibold">
                    {plan.router_included ? t('services.yes') : t('services.no')}
                  </span>
                </div>
              </div>

              {/* Cabinet notes indicator */}
              {plan.cabinet_notes && (
                <div className="flex items-center gap-1.5 text-xs text-amber-400 mt-3 bg-amber-500/5 p-1.5 rounded-lg border border-amber-500/10 justify-center w-full">
                  <StickyNote className="w-3.5 h-3.5" />
                  <span>{t('services.has_cabinet_notes')}</span>
                </div>
              )}

              {/* Manager actions */}
              {manager && (
                <div className="absolute top-4 left-4 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={e => e.stopPropagation()}>
                  <button onClick={() => setEditPlan(plan)}
                    className="p-1.5 rounded-lg bg-background/80 hover:bg-background border border-border/50 text-muted-foreground hover:text-primary transition-colors">
                    <Pencil className="w-3.5 h-3.5" />
                  </button>
                  <button onClick={() => setDelPlan(plan)}
                    className="p-1.5 rounded-lg bg-background/80 hover:bg-background border border-border/50 text-muted-foreground hover:text-destructive transition-colors">
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Detail Modal */}
      {detailPlan && <PlanDetailModal plan={detailPlan} onClose={() => setDetailPlan(null)} />}

      {/* Edit/Create Modal */}
      {editPlan !== undefined && (
        <PlanModal
          categoryId={category.id}
          initial={editPlan}
          onClose={() => setEditPlan(undefined)}
          onSaved={() => qc.invalidateQueries({ queryKey: key })}
        />
      )}

      {/* Delete Confirm */}
      {delPlan && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-card border border-border rounded-2xl p-6 w-full max-w-sm shadow-2xl text-center space-y-4">
            <Trash2 className="w-10 h-10 text-destructive mx-auto" />
            <p className="font-bold text-foreground">{t('services.delete_plan_title', { name: delPlan.name })}</p>
            <p className="text-muted-foreground text-sm">{t('services.delete_plan_desc')}</p>
            <div className="flex gap-3">
              <button onClick={() => setDelPlan(null)}
                className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 text-sm font-medium">
                {t('services.cancel')}
              </button>
              <button onClick={() => deleteMut.mutate(delPlan.id)} disabled={deleteMut.isPending}
                className="flex-1 py-2.5 rounded-xl bg-destructive text-destructive-foreground hover:bg-destructive/90 text-sm font-medium flex items-center justify-center gap-2">
                {deleteMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : null} {t('services.delete')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
