import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import type { ServiceCategory, ServicePlan } from './types';
import { IRAQ_PROVINCES } from './types';
import {
  ArrowRight, Plus, Pencil, Trash2, Wifi, Loader2, X,
  Clock, DollarSign, Zap, MapPin, Server, Router, Cpu, StickyNote,
} from 'lucide-react';

/* ─── Plan Detail Modal ────────────────────────────────── */
function PlanDetailModal({ plan, onClose }: { plan: ServicePlan; onClose: () => void }) {
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
              { icon: DollarSign, label: 'السعر الشهري', value: `${plan.price.toLocaleString()} د.ع`, color: 'text-emerald-400' },
              { icon: Clock, label: 'مدة الباقة', value: `${plan.duration_days} يوم`, color: 'text-blue-400' },
              { icon: MapPin, label: 'المحافظة', value: plan.province, color: 'text-amber-400' },
              { icon: Zap, label: 'سرعة التحميل', value: plan.speed_download ?? '—', color: 'text-primary' },
              { icon: Zap, label: 'سرعة الرفع', value: plan.speed_upload ?? '—', color: 'text-primary' },
              { icon: Server, label: 'نوع IP', value: plan.ip_type, color: 'text-purple-400' },
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
              <span>نوع الاتصال: <span className="text-foreground font-medium">{plan.connection_type}</span></span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Server className="w-4 h-4 shrink-0" />
              <span>الحد الأقصى للبيانات: <span className="text-foreground font-medium">{plan.data_cap ?? 'غير محدود'}</span></span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <DollarSign className="w-4 h-4 shrink-0" />
              <span>رسوم التركيب: <span className="text-foreground font-medium">{plan.installation_fee > 0 ? `${plan.installation_fee.toLocaleString()} د.ع` : 'مجاني'}</span></span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Router className="w-4 h-4 shrink-0" />
              <span>راوتر مشمول: <span className="text-foreground font-medium">{plan.router_included ? 'نعم ✓' : 'لا'}</span></span>
            </div>
          </div>

          {/* Description */}
          {plan.description && (
            <div className="bg-background rounded-xl p-4 border border-border">
              <p className="text-xs text-muted-foreground mb-1">تفاصيل إضافية</p>
              <p className="text-foreground text-sm whitespace-pre-line">{plan.description}</p>
            </div>
          )}

          {/* Cabinet Notes */}
          {plan.cabinet_notes && (
            <div className="bg-amber-500/5 rounded-xl p-4 border border-amber-500/20">
              <div className="flex items-center gap-2 mb-2">
                <StickyNote className="w-4 h-4 text-amber-400" />
                <p className="text-amber-400 text-xs font-semibold">ملاحظات الكابينات المشمولة</p>
              </div>
              <p className="text-foreground text-sm whitespace-pre-line leading-relaxed">{plan.cabinet_notes}</p>
            </div>
          )}

          <div className="flex items-center justify-between text-xs text-muted-foreground pt-2 border-t border-border">
            <span>أُنشئت بواسطة: {plan.creator_name ?? '—'}</span>
            <span className={`px-2 py-0.5 rounded-full font-medium ${plan.is_active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-destructive/10 text-destructive'}`}>
              {plan.is_active ? 'نشطة' : 'معطّلة'}
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
  const [f, setF] = useState({
    name: initial?.name ?? '',
    price: initial?.price?.toString() ?? '',
    duration_days: initial?.duration_days?.toString() ?? '30',
    province: initial?.province ?? '',
    speed_download: initial?.speed_download ?? '',
    speed_upload: initial?.speed_upload ?? '',
    data_cap: initial?.data_cap ?? 'Unlimited',
    connection_type: initial?.connection_type ?? 'FTTH',
    installation_fee: initial?.installation_fee?.toString() ?? '0',
    router_included: initial?.router_included ?? false,
    ip_type: initial?.ip_type ?? 'Dynamic',
    description: initial?.description ?? '',
    cabinet_notes: initial?.cabinet_notes ?? '',
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');

  const set = (k: string, v: any) => setF(p => ({ ...p, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!f.name.trim() || !f.price || !f.province) return setErr('الاسم والسعر والمحافظة مطلوبة');
    setBusy(true); setErr('');
    try {
      const payload = {
        ...f,
        price: parseFloat(f.price),
        duration_days: parseInt(f.duration_days),
        installation_fee: parseFloat(f.installation_fee),
        speed_download: f.speed_download || undefined,
        speed_upload: f.speed_upload || undefined,
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
      setErr(e.response?.data?.error ?? 'حدث خطأ');
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
          <h2 className="font-bold text-foreground">{initial ? 'تعديل الباقة' : 'باقة جديدة'}</h2>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>
        <form onSubmit={submit} className="p-6 overflow-y-auto space-y-4">
          <div className="grid grid-cols-2 gap-4">
            {field('اسم الباقة *', 'name', 'text', 'مثال: باقة 100 ميغا')}
            {field('السعر (د.ع) *', 'price', 'number', '50000')}
            {field('المدة (أيام) *', 'duration_days', 'number', '30')}
            {field('سرعة التحميل', 'speed_download', 'text', '100 Mbps')}
            {field('سرعة الرفع', 'speed_upload', 'text', '50 Mbps')}
            {field('الحد الأقصى للبيانات', 'data_cap', 'text', 'Unlimited')}
            {field('رسوم التركيب (د.ع)', 'installation_fee', 'number', '0')}
          </div>

          {/* Province */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">المحافظة *</label>
            <select value={f.province} onChange={e => set('province', e.target.value)}
              className="w-full bg-background border border-border rounded-xl px-3 py-2 text-sm text-foreground focus:outline-none focus:border-primary/60">
              <option value="">اختر المحافظة...</option>
              {IRAQ_PROVINCES.map(p => <option key={p} value={p}>{p}</option>)}
            </select>
          </div>

          <div className="grid grid-cols-2 gap-4">
            {/* IP Type */}
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-1">نوع IP</label>
              <select value={f.ip_type} onChange={e => set('ip_type', e.target.value)}
                className="w-full bg-background border border-border rounded-xl px-3 py-2 text-sm text-foreground focus:outline-none focus:border-primary/60">
                <option value="Dynamic">Dynamic</option>
                <option value="Static">Static</option>
              </select>
            </div>
            {/* Router included */}
            <div className="flex items-center gap-3 pt-5">
              <input type="checkbox" id="router" checked={f.router_included}
                onChange={e => set('router_included', e.target.checked)}
                className="w-4 h-4 accent-primary" />
              <label htmlFor="router" className="text-sm text-foreground">راوتر مشمول مع الباقة</label>
            </div>
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">وصف إضافي</label>
            <textarea value={f.description} onChange={e => set('description', e.target.value)}
              rows={2} placeholder="تفاصيل الباقة..."
              className="w-full bg-background border border-border rounded-xl px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60 resize-none" />
          </div>

          {/* Cabinet Notes */}
          <div>
            <label className="block text-xs font-medium text-amber-400 mb-1 flex items-center gap-1.5">
              <StickyNote className="w-3.5 h-3.5" /> ملاحظات الكابينات المشمولة
            </label>
            <textarea value={f.cabinet_notes} onChange={e => set('cabinet_notes', e.target.value)}
              rows={3} placeholder="اكتب أسماء أو أرقام الكابينات المشمولة بهذا العرض..."
              className="w-full bg-amber-500/5 border border-amber-500/20 rounded-xl px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-amber-500/40 resize-none" />
          </div>

          {err && <p className="text-destructive text-sm">{err}</p>}

          <div className="flex gap-3 pt-2 sticky bottom-0 bg-card pb-1">
            <button type="button" onClick={onClose}
              className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 text-sm font-medium">
              إلغاء
            </button>
            <button type="submit" disabled={busy}
              className="flex-1 py-2.5 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium flex items-center justify-center gap-2">
              {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
              {initial ? 'حفظ' : 'إنشاء الباقة'}
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
  const qc = useQueryClient();
  const key = ['service-plans', category.id];

  const [detailPlan, setDetailPlan] = useState<ServicePlan | null>(null);
  const [editPlan, setEditPlan] = useState<ServicePlan | null | undefined>(undefined);
  const [delPlan, setDelPlan] = useState<ServicePlan | null>(null);

  const { data: plans, isLoading } = useQuery<ServicePlan[]>({
    queryKey: key,
    queryFn: async () => (await api.get(`/services/categories/${category.id}/plans`)).data.data ?? [],
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/services/plans/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: key }); setDelPlan(null); },
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button onClick={onBack}
          className="p-2 rounded-xl border border-border hover:bg-white/5 transition-colors">
          <ArrowRight className="w-5 h-5 text-muted-foreground" />
        </button>
        <div className="flex-1">
          <h1 className="text-xl font-bold text-foreground flex items-center gap-2">
            <Wifi className="w-5 h-5 text-primary" /> {category.name}
          </h1>
          {category.description && (
            <p className="text-muted-foreground text-sm">{category.description}</p>
          )}
        </div>
        {manager && (
          <button onClick={() => setEditPlan(null)}
            className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-all shadow-lg shadow-primary/20">
            <Plus className="w-4 h-4" /> باقة جديدة
          </button>
        )}
      </div>

      {/* Plans grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="w-8 h-8 text-primary animate-spin" />
        </div>
      ) : !plans?.length ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="w-16 h-16 rounded-2xl bg-muted/30 flex items-center justify-center mb-4">
            <Wifi className="w-8 h-8 text-muted-foreground" />
          </div>
          <p className="font-semibold text-foreground">لا توجد باقات في هذا التصنيف</p>
          <p className="text-muted-foreground text-sm mt-1">
            {manager ? 'اضغط "باقة جديدة" لإضافة أول باقة' : 'لا توجد باقات متاحة حالياً'}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {plans.map(plan => (
            <div key={plan.id}
              className="group relative bg-card border border-border rounded-2xl p-5 hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5 transition-all duration-200 cursor-pointer"
              onClick={() => setDetailPlan(plan)}>

              {/* Province badge */}
              <div className="flex items-center justify-between mb-4">
                <span className="flex items-center gap-1 text-xs px-2.5 py-1 rounded-full bg-amber-500/10 text-amber-400 font-medium">
                  <MapPin className="w-3 h-3" /> {plan.province}
                </span>
                {!plan.is_active && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-destructive/10 text-destructive">معطّلة</span>
                )}
              </div>

              {/* Name */}
              <h3 className="font-bold text-foreground text-base mb-3">{plan.name}</h3>

              {/* Stats row */}
              <div className="grid grid-cols-2 gap-2 mb-4">
                <div className="bg-background rounded-xl p-2.5 text-center">
                  <p className="text-emerald-400 font-bold text-sm">{plan.price.toLocaleString()}</p>
                  <p className="text-muted-foreground text-xs">د.ع / الباقة</p>
                </div>
                <div className="bg-background rounded-xl p-2.5 text-center">
                  <p className="text-blue-400 font-bold text-sm">{plan.duration_days}</p>
                  <p className="text-muted-foreground text-xs">يوم</p>
                </div>
              </div>

              {/* Speed */}
              {(plan.speed_download || plan.speed_upload) && (
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Zap className="w-3.5 h-3.5 text-primary" />
                  {plan.speed_download && <span>{plan.speed_download}↓</span>}
                  {plan.speed_upload && <span>{plan.speed_upload}↑</span>}
                </div>
              )}

              {/* Cabinet notes indicator */}
              {plan.cabinet_notes && (
                <div className="flex items-center gap-1.5 text-xs text-amber-400 mt-2">
                  <StickyNote className="w-3.5 h-3.5" />
                  <span>يحتوي ملاحظات كابينات</span>
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
            <p className="font-bold text-foreground">حذف الباقة "{delPlan.name}"؟</p>
            <p className="text-muted-foreground text-sm">لا يمكن التراجع عن هذا الإجراء.</p>
            <div className="flex gap-3">
              <button onClick={() => setDelPlan(null)}
                className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 text-sm font-medium">
                إلغاء
              </button>
              <button onClick={() => deleteMut.mutate(delPlan.id)} disabled={deleteMut.isPending}
                className="flex-1 py-2.5 rounded-xl bg-destructive text-destructive-foreground hover:bg-destructive/90 text-sm font-medium flex items-center justify-center gap-2">
                {deleteMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : null} حذف
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
