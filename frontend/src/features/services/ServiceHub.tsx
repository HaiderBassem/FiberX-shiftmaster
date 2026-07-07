import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/store/authStore';
import api from '@/lib/api';
import type { ServiceCategory } from './types';
import { ServicePlans } from './ServicePlans';
import {
  Plus, Pencil, Trash2, Wifi, FolderOpen, ChevronRight, X, Loader2, ToggleLeft, ToggleRight,
} from 'lucide-react';

/* ─── helpers ─────────────────────────────────────────── */
const canManage = (user: any) =>
  user?.role === 'admin' ||
  (user?.role === 'team_leader' && user?.can_manage_services === true);

/* ─── Category Form Modal ──────────────────────────────── */
function CategoryModal({
  initial, onClose, onSaved,
}: {
  initial?: ServiceCategory | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(initial?.name ?? '');
  const [desc, setDesc] = useState(initial?.description ?? '');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return setErr('الاسم مطلوب');
    setBusy(true); setErr('');
    try {
      const payload = { name: name.trim(), description: desc.trim() || undefined };
      if (initial) {
        await api.put(`/services/categories/${initial.id}`, { ...payload, is_active: initial.is_active });
      } else {
        await api.post('/services/categories', payload);
      }
      onSaved();
      onClose();
    } catch (e: any) {
      setErr(e.response?.data?.error ?? 'حدث خطأ');
    } finally { setBusy(false); }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-card border border-border rounded-2xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between px-6 py-5 border-b border-border">
          <h2 className="text-lg font-bold text-foreground">
            {initial ? 'تعديل التصنيف' : 'تصنيف جديد'}
          </h2>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>
        <form onSubmit={submit} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">اسم التصنيف *</label>
            <input
              value={name} onChange={e => setName(e.target.value)}
              placeholder="مثال: عروض بغداد الشمالية"
              className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60 transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">وصف (اختياري)</label>
            <textarea
              value={desc} onChange={e => setDesc(e.target.value)} rows={3}
              placeholder="وصف مختصر لهذا التصنيف..."
              className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60 transition-colors resize-none"
            />
          </div>
          {err && <p className="text-destructive text-sm">{err}</p>}
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={onClose}
              className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 transition-colors text-sm font-medium">
              إلغاء
            </button>
            <button type="submit" disabled={busy}
              className="flex-1 py-2.5 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 transition-colors text-sm font-medium flex items-center justify-center gap-2">
              {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
              {initial ? 'حفظ التعديلات' : 'إنشاء'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

/* ─── Main ServiceHub ──────────────────────────────────── */
export function ServiceHub() {
  const { user } = useAuthStore();
  const qc = useQueryClient();
  const manager = canManage(user);

  const [openCatID, setOpenCatID] = useState<string | null>(null);
  const [modalCat, setModalCat] = useState<ServiceCategory | null | undefined>(undefined); // undefined=closed
  const [delConfirm, setDelConfirm] = useState<ServiceCategory | null>(null);

  const { data, isLoading } = useQuery<ServiceCategory[]>({
    queryKey: ['service-categories'],
    queryFn: async () => (await api.get('/services/categories')).data.data ?? [],
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/services/categories/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['service-categories'] }); setDelConfirm(null); },
  });

  const toggleMut = useMutation({
    mutationFn: (cat: ServiceCategory) =>
      api.put(`/services/categories/${cat.id}`, { name: cat.name, description: cat.description, is_active: !cat.is_active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['service-categories'] }),
  });

  if (openCatID) {
    const cat = data?.find(c => c.id === openCatID);
    return (
      <ServicePlans
        category={cat!}
        manager={manager}
        onBack={() => setOpenCatID(null)}
      />
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
              <Wifi className="w-5 h-5 text-primary" />
            </div>
            خدمات الإنترنت
          </h1>
          <p className="text-muted-foreground mt-1 text-sm">
            كتالوج باقات FTTH — الأسعار والمحافظات والكابينات
          </p>
        </div>
        {manager && (
          <button
            onClick={() => setModalCat(null)}
            className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-all shadow-lg shadow-primary/20"
          >
            <Plus className="w-4 h-4" /> تصنيف جديد
          </button>
        )}
      </div>

      {/* Grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="w-8 h-8 text-primary animate-spin" />
        </div>
      ) : !data?.length ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="w-16 h-16 rounded-2xl bg-muted/30 flex items-center justify-center mb-4">
            <FolderOpen className="w-8 h-8 text-muted-foreground" />
          </div>
          <p className="text-foreground font-semibold text-lg">لا توجد تصنيفات بعد</p>
          <p className="text-muted-foreground text-sm mt-1">
            {manager ? 'اضغط "تصنيف جديد" لإضافة أول تصنيف' : 'لا توجد خدمات متاحة حالياً'}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
          {data.map(cat => (
            <div
              key={cat.id}
              className="group relative bg-card border border-border rounded-2xl p-5 hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5 transition-all duration-200 cursor-pointer"
              onClick={() => setOpenCatID(cat.id)}
            >
              {/* Icon */}
              <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center mb-4 group-hover:from-primary/30 transition-colors">
                <Wifi className="w-6 h-6 text-primary" />
              </div>

              {/* Name & desc */}
              <h3 className="font-bold text-foreground text-base leading-snug mb-1">{cat.name}</h3>
              {cat.description && (
                <p className="text-muted-foreground text-xs line-clamp-2 mb-3">{cat.description}</p>
              )}

              {/* Plan count badge */}
              <div className="flex items-center gap-1.5 mt-auto">
                <span className="text-xs px-2 py-0.5 rounded-full bg-primary/10 text-primary font-medium">
                  {cat.plan_count} باقة
                </span>
                {!cat.is_active && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-destructive/10 text-destructive font-medium">
                    معطّل
                  </span>
                )}
              </div>

              <ChevronRight className="absolute bottom-5 left-5 w-4 h-4 text-muted-foreground/40 group-hover:text-primary transition-colors" />

              {/* Actions — stop propagation */}
              {manager && (
                <div
                  className="absolute top-4 left-4 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={e => e.stopPropagation()}
                >
                  <button
                    onClick={() => toggleMut.mutate(cat)}
                    className="p-1.5 rounded-lg bg-background/80 hover:bg-background border border-border/50 text-muted-foreground hover:text-primary transition-colors"
                    title={cat.is_active ? 'تعطيل' : 'تفعيل'}
                  >
                    {cat.is_active
                      ? <ToggleRight className="w-3.5 h-3.5 text-primary" />
                      : <ToggleLeft className="w-3.5 h-3.5" />}
                  </button>
                  <button
                    onClick={() => setModalCat(cat)}
                    className="p-1.5 rounded-lg bg-background/80 hover:bg-background border border-border/50 text-muted-foreground hover:text-primary transition-colors"
                  >
                    <Pencil className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => setDelConfirm(cat)}
                    className="p-1.5 rounded-lg bg-background/80 hover:bg-background border border-border/50 text-muted-foreground hover:text-destructive transition-colors"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Category Modal */}
      {modalCat !== undefined && (
        <CategoryModal
          initial={modalCat}
          onClose={() => setModalCat(undefined)}
          onSaved={() => qc.invalidateQueries({ queryKey: ['service-categories'] })}
        />
      )}

      {/* Delete Confirm */}
      {delConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-card border border-border rounded-2xl p-6 w-full max-w-sm shadow-2xl text-center space-y-4">
            <div className="w-12 h-12 rounded-xl bg-destructive/10 flex items-center justify-center mx-auto">
              <Trash2 className="w-6 h-6 text-destructive" />
            </div>
            <div>
              <p className="font-bold text-foreground">حذف التصنيف؟</p>
              <p className="text-muted-foreground text-sm mt-1">
                سيتم حذف <span className="font-semibold text-foreground">"{delConfirm.name}"</span> وجميع باقاته. لا يمكن التراجع.
              </p>
            </div>
            <div className="flex gap-3">
              <button onClick={() => setDelConfirm(null)}
                className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 transition-colors text-sm font-medium">
                إلغاء
              </button>
              <button
                onClick={() => deleteMut.mutate(delConfirm.id)}
                disabled={deleteMut.isPending}
                className="flex-1 py-2.5 rounded-xl bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors text-sm font-medium flex items-center justify-center gap-2">
                {deleteMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                حذف
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
