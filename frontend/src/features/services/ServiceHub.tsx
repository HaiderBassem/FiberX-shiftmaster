import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/store/authStore';
import api from '@/lib/api';
import type { ServiceCategory, Province } from './types';
import { ServicePlans } from './ServicePlans';
import { ProvinceManager } from './ProvinceManager';
import { ProvinceShareModal } from './ProvinceShareModal';
import { useTranslation } from 'react-i18next';
import {
  Plus, Pencil, Trash2, Wifi, FolderOpen, ChevronRight, X, Loader2, ToggleLeft, ToggleRight, Search, MapPin, ArrowRight, Share2, GripVertical
} from 'lucide-react';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  rectSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

/* ─── helpers ─────────────────────────────────────────── */
const canManage = (user: any) =>
  user?.role === 'admin' ||
  ((user?.role === 'team_leader' || user?.role === 'manager') && user?.can_manage_services === true);

/* ─── Category Form Modal ──────────────────────────────── */
function CategoryModal({
  initial, provinceId, onClose, onSaved,
}: {
  initial?: ServiceCategory | null;
  provinceId: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t } = useTranslation();
  const [name, setName] = useState(initial?.name ?? '');
  const [desc, setDesc] = useState(initial?.description ?? '');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return setErr(t('services.name_required'));
    setBusy(true); setErr('');
    try {
      const payload = { province_id: provinceId, name: name.trim(), description: desc.trim() || undefined };
      if (initial) {
        await api.put(`/services/categories/${initial.id}`, { ...payload, is_active: initial.is_active });
      } else {
        await api.post('/services/categories', payload);
      }
      onSaved();
      onClose();
    } catch (e: any) {
      setErr(e.response?.data?.error ?? t('services.error_occurred'));
    } finally { setBusy(false); }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-card border border-border rounded-2xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between px-6 py-5 border-b border-border">
          <h2 className="text-lg font-bold text-foreground">
            {initial ? t('services.edit_category') : t('services.new_category')}
          </h2>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>
        <form onSubmit={submit} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">{t('services.category_name')}</label>
            <input
              value={name} onChange={e => setName(e.target.value)}
              placeholder={t('services.category_name_ph')}
              className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60 transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">{t('services.category_desc')}</label>
            <textarea
              value={desc} onChange={e => setDesc(e.target.value)} rows={3}
              placeholder={t('services.category_desc_ph')}
              className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/60 transition-colors resize-none"
            />
          </div>
          {err && <p className="text-destructive text-sm">{err}</p>}
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={onClose}
              className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 transition-colors text-sm font-medium">
              {t('services.cancel')}
            </button>
            <button type="submit" disabled={busy}
              className="flex-1 py-2.5 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 transition-colors text-sm font-medium flex items-center justify-center gap-2">
              {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
              {initial ? t('services.save_changes') : t('services.create')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

/* ─── Sortable Category Card ─────────────────────────────── */
function SortableCategoryCard({
  cat, manager, province, t, onSelectCategory, setDelConfirm, setModalCat, toggleMut, isDragEnabled
}: {
  cat: ServiceCategory; manager: boolean; province: Province; t: any;
  onSelectCategory: (cat: ServiceCategory) => void;
  setDelConfirm: (cat: ServiceCategory) => void;
  setModalCat: (cat: ServiceCategory) => void;
  toggleMut: any;
  isDragEnabled: boolean;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: cat.id, disabled: !isDragEnabled });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 10 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`group relative border rounded-2xl p-5 transition-all duration-200 cursor-pointer ${
        cat.is_active 
          ? 'bg-card border-border hover:border-primary/40' 
          : 'bg-muted/10 border-border/50 opacity-60 hover:opacity-100 grayscale-[30%] hover:grayscale-0'
      } ${isDragging ? 'opacity-50 shadow-2xl scale-105' : 'hover:shadow-lg hover:shadow-primary/5'}`}
      onClick={() => onSelectCategory(cat)}
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
      <div className="flex flex-col gap-2 mt-auto">
        <div className="flex items-center gap-1.5">
          <span className="text-xs px-2 py-0.5 rounded-full bg-primary/10 text-primary font-medium">
            {t('services.plan_count', { count: cat.plan_count || 0 })}
          </span>
          {!cat.is_active && (
            <span className="text-xs px-2 py-0.5 rounded-full bg-destructive/10 text-destructive font-medium">
              {t('services.disabled')}
            </span>
          )}
        </div>
        {!cat.is_active && cat.disabled_at && (
          <span className="text-[10px] text-muted-foreground/70">
            Stopped: {new Date(cat.disabled_at).toLocaleDateString()}
          </span>
        )}
      </div>

      <ChevronRight className="absolute bottom-5 left-5 w-4 h-4 text-muted-foreground/40 group-hover:text-primary transition-colors" />

      {/* Actions */}
      {manager && !province.is_shared && (
        <div
          className={`absolute top-4 left-4 flex gap-1 transition-opacity ${isDragging ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'}`}
          onClick={e => e.stopPropagation()}
        >
          {/* Drag Handle */}
          {isDragEnabled && (
            <div {...listeners} {...attributes} className="p-1.5 cursor-grab active:cursor-grabbing hover:bg-background border border-transparent hover:border-border/50 rounded-lg text-muted-foreground transition-colors mr-1">
              <GripVertical className="w-4 h-4" />
            </div>
          )}
          <button
            onClick={() => toggleMut.mutate(cat)}
            className="p-1.5 rounded-lg bg-background/80 hover:bg-background border border-border/50 text-muted-foreground hover:text-primary transition-colors"
            title={cat.is_active ? t('services.disable') : t('services.enable')}
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
  );
}

/* ─── Categories View ──────────────────────────────────── */
function CategoriesView({
  province, manager, onBack, onSelectCategory
}: {
  province: Province; manager: boolean; onBack: () => void; onSelectCategory: (cat: ServiceCategory) => void;
}) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [modalCat, setModalCat] = useState<ServiceCategory | null | undefined>(undefined);
  const [delConfirm, setDelConfirm] = useState<ServiceCategory | null>(null);

  const queryKey = ['service-categories', province.id];

  const { data: categories, isLoading } = useQuery<ServiceCategory[]>({
    queryKey,
    queryFn: async () => (await api.get(`/services/categories?province_id=${province.id}`)).data.data ?? [],
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/services/categories/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey }); setDelConfirm(null); },
  });

  const toggleMut = useMutation({
    mutationFn: async (cat: ServiceCategory) => {
      return api.put(`/services/categories/${cat.id}`, {
        name: cat.name,
        description: cat.description,
        is_active: !cat.is_active
      });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey })
  });

  const reorderMut = useMutation({
    mutationFn: async (categoryIds: string[]) => {
      return api.put('/services/categories/reorder', { category_ids: categoryIds });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey })
  });

  const filteredCategories = (categories ?? []).filter(cat => 
    cat.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (cat.description && cat.description.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = filteredCategories.findIndex(c => c.id === active.id);
      const newIndex = filteredCategories.findIndex(c => c.id === over.id);
      const newCats = arrayMove(filteredCategories, oldIndex, newIndex);
      reorderMut.mutate(newCats.map(c => c.id));
    }
  };

  return (
    <div className="space-y-8">
      {/* Province Info Banner */}
      <div className="flex flex-wrap items-center justify-between gap-4 bg-primary/5 border border-primary/10 rounded-2xl p-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
            <MapPin className="w-5 h-5 text-primary" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('services.province')}</p>
            <h2 className="font-bold text-foreground text-lg">{province.name}</h2>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <button
            onClick={onBack}
            className="flex items-center gap-2 px-3 py-2 rounded-xl border border-border bg-card hover:bg-white/5 text-xs font-medium transition-colors"
          >
            <ArrowRight className="w-4 h-4" /> {t('services.back_to_provinces')}
          </button>
        </div>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
              <Wifi className="w-5 h-5 text-primary" />
            </div>
            {t('services.title')}
          </h1>
          <p className="text-muted-foreground mt-1 text-sm">
            {t('services.desc')}
          </p>
        </div>
        {manager && !province.is_shared && (
          <button
            onClick={() => setModalCat(null)}
            className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-all shadow-lg shadow-primary/20"
          >
            <Plus className="w-4 h-4" /> {t('services.new_category')}
          </button>
        )}
      </div>

      {/* Search Bar */}
      <div className="relative max-w-md w-full mb-6">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t('services.search_categories_placeholder')}
          className="w-full pl-9 pr-4 py-2 bg-card border border-border rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-primary/20"
        />
      </div>

      {/* Grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="w-8 h-8 text-primary animate-spin" />
        </div>
      ) : !filteredCategories.length ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="w-16 h-16 rounded-2xl bg-muted/30 flex items-center justify-center mb-4">
            <FolderOpen className="w-8 h-8 text-muted-foreground" />
          </div>
          <p className="text-foreground font-semibold text-lg">{t('services.no_categories')}</p>
          <p className="text-muted-foreground text-sm mt-1">
            {manager ? t('services.no_categories_desc_manager') : t('services.no_categories_desc_employee')}
          </p>
        </div>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={filteredCategories.map(c => c.id)} strategy={rectSortingStrategy}>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
              {filteredCategories.map((cat) => (
                <SortableCategoryCard
                  key={cat.id}
                  cat={cat}
                  manager={manager}
                  province={province}
                  t={t}
                  onSelectCategory={onSelectCategory}
                  setDelConfirm={setDelConfirm}
                  setModalCat={setModalCat}
                  toggleMut={toggleMut}
                  isDragEnabled={!searchQuery}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}


      {/* Category Modal */}
      {modalCat !== undefined && (
        <CategoryModal
          initial={modalCat}
          provinceId={province.id}
          onClose={() => setModalCat(undefined)}
          onSaved={() => qc.invalidateQueries({ queryKey: ['service-categories', province.id] })}
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
              <p className="font-bold text-foreground">{t('services.delete_category_title')}</p>
              <p className="text-muted-foreground text-sm mt-1">
                {t('services.delete_category_desc', { name: delConfirm.name })}
              </p>
            </div>
            <div className="flex gap-3">
              <button onClick={() => setDelConfirm(null)}
                className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground hover:bg-white/5 transition-colors text-sm font-medium">
                {t('services.cancel')}
              </button>
              <button
                onClick={() => deleteMut.mutate(delConfirm.id)}
                disabled={deleteMut.isPending}
                className="flex-1 py-2.5 rounded-xl bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors text-sm font-medium flex items-center justify-center gap-2">
                {deleteMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                {t('services.delete')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/* ─── Main ServiceHub ──────────────────────────────────── */
export function ServiceHub() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const manager = canManage(user);

  const [selectedProvince, setSelectedProvince] = useState<Province | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<ServiceCategory | null>(null);
  const [provinceSearchQuery, setProvinceSearchQuery] = useState('');
  const [showProvManager, setShowProvManager] = useState(false);
  const [shareProv, setShareProv] = useState<Province | null>(null);

  const { data: provincesData, isLoading: isProvLoading } = useQuery<Province[]>({
    queryKey: ['provinces'],
    queryFn: async () => (await api.get('/provinces')).data.data ?? [],
  });

  // Step 3: Show Plans
  if (selectedCategory && selectedProvince) {
    return (
      <ServicePlans
        category={selectedCategory}
        manager={manager && !selectedProvince.is_shared}
        onBack={() => setSelectedCategory(null)}
      />
    );
  }

  // Step 2: Show Categories
  if (selectedProvince) {
    return (
      <CategoriesView
        province={selectedProvince}
        manager={manager}
        onBack={() => setSelectedProvince(null)}
        onSelectCategory={setSelectedCategory}
      />
    );
  }

  // Step 1: Select Province
  const filteredProvinces = (provincesData ?? []).filter(p =>
    p.name.toLowerCase().includes(provinceSearchQuery.toLowerCase())
  );

  return (
    <div className="space-y-8">
      {showProvManager && <ProvinceManager onClose={() => setShowProvManager(false)} />}
      {shareProv && <ProvinceShareModal province={shareProv} onClose={() => setShareProv(null)} />}
      
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-foreground flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
              <Wifi className="w-5 h-5 text-primary" />
            </div>
            {t('services.title')}
          </h1>
          <p className="text-muted-foreground mt-1 text-sm">
            {t('services.select_province_desc')}
          </p>
        </div>
        {manager && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowProvManager(true)}
              className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-border bg-card text-foreground text-sm font-medium hover:bg-muted transition-all"
            >
              <MapPin className="w-4 h-4" /> {t('services.manage_provinces')}
            </button>
          </div>
        )}
      </div>

      {/* Search Bar */}
      <div className="relative max-w-md w-full mb-6">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <input
          type="text"
          value={provinceSearchQuery}
          onChange={(e) => setProvinceSearchQuery(e.target.value)}
          placeholder={t('services.search_provinces_placeholder')}
          className="w-full pl-9 pr-4 py-2 bg-card border border-border rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-primary/20"
        />
      </div>

      {/* Provinces Grid */}
      {isProvLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="w-8 h-8 text-primary animate-spin" />
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
          {filteredProvinces.map(p => (
            <button
              key={p.id}
              onClick={() => setSelectedProvince(p)}
              className={`group relative flex flex-col items-center justify-center gap-3 p-6 rounded-2xl border transition-all duration-200 ${
                p.is_active 
                  ? 'bg-card border-border hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5' 
                  : 'bg-muted/10 border-border/50 opacity-60 hover:opacity-100 grayscale-[30%] hover:grayscale-0 hover:shadow-lg'
              }`}
            >
              <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center group-hover:scale-110 transition-transform duration-200 group-hover:bg-primary/20">
                <MapPin className="w-6 h-6 text-primary" />
              </div>
              <div className="flex flex-col items-center gap-1">
                <span className="font-semibold text-foreground">{p.name}</span>
                {!p.is_active && (
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-destructive/10 text-destructive font-medium">
                    {t('services.disabled')}
                  </span>
                )}
              </div>
              {p.is_shared && (
                <div className="absolute top-2 right-2 text-blue-500">
                  <Share2 className="w-4 h-4" />
                </div>
              )}
              {manager && !p.is_shared && (
                <div 
                  className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={e => { e.stopPropagation(); setShareProv(p); }}
                >
                  <div className="p-1.5 rounded-lg bg-background/80 hover:bg-background border border-border/50 text-muted-foreground hover:text-blue-500 transition-colors" title={t('services.share_province')}>
                    <Share2 className="w-4 h-4" />
                  </div>
                </div>
              )}
            </button>
          ))}
          {filteredProvinces.length === 0 && (
            <div className="col-span-full py-12 text-center text-muted-foreground">
              {t('services.no_provinces_found')}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
