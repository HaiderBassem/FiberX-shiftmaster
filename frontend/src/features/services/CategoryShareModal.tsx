import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { X, Loader2, Share2, Plus, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { ServiceCategory, ServiceCategoryShare } from './types';

export function CategoryShareModal({ category, onClose }: { category: ServiceCategory; onClose: () => void }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [selectedDept, setSelectedDept] = useState('');

  const { data: shares, isLoading: loadingShares } = useQuery<ServiceCategoryShare[]>({
    queryKey: ['category-shares', category.id],
    queryFn: async () => (await api.get(`/services/categories/${category.id}/shares`)).data.data ?? [],
  });

  const { data: allDepts } = useQuery({
    queryKey: ['departments'],
    queryFn: async () => (await api.get('/departments')).data.data ?? [],
  });

  const shareMut = useMutation({
    mutationFn: (departmentId: string) => api.post(`/services/categories/${category.id}/shares`, { department_id: departmentId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['category-shares', category.id] });
      setSelectedDept('');
    },
  });

  const removeMut = useMutation({
    mutationFn: (departmentId: string) => api.delete(`/services/categories/${category.id}/shares/${departmentId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['category-shares', category.id] }),
  });

  // Filter out departments that already have shares
  const availableDepts = allDepts?.filter((d: any) => 
    !shares?.find(s => s.department_id === d.id) && d.id !== category.department_id
  ) ?? [];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-card border border-border rounded-2xl w-full max-w-lg shadow-2xl flex flex-col max-h-[90vh]">
        <div className="flex items-center justify-between px-6 py-5 border-b border-border">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
              <Share2 className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-foreground">{t('services.share_category')}</h2>
              <p className="text-xs text-muted-foreground">{category.name}</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>

        <div className="p-6 overflow-y-auto space-y-6">
          <div className="flex items-center gap-3">
            <select
              value={selectedDept}
              onChange={e => setSelectedDept(e.target.value)}
              className="flex-1 bg-background border border-border rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="">{t('services.select_department_to_share')}</option>
              {availableDepts.map((d: any) => (
                <option key={d.id} value={d.id}>{d.name}</option>
              ))}
            </select>
            <button
              onClick={() => { if (selectedDept) shareMut.mutate(selectedDept); }}
              disabled={!selectedDept || shareMut.isPending}
              className="px-4 py-2.5 bg-primary text-primary-foreground text-sm font-medium rounded-xl hover:bg-primary/90 disabled:opacity-50 flex items-center gap-2 transition-all"
            >
              {shareMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
              {t('services.share')}
            </button>
          </div>

          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-foreground">{t('services.shared_with_departments')}</h3>
            {loadingShares ? (
              <div className="flex justify-center py-4"><Loader2 className="w-5 h-5 animate-spin text-primary" /></div>
            ) : shares?.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-4">{t('services.not_shared_yet')}</p>
            ) : (
              <div className="space-y-2">
                {shares?.map(s => (
                  <div key={s.id} className="flex items-center justify-between p-3 bg-background border border-border rounded-xl">
                    <span className="text-sm font-medium">{s.department_name}</span>
                    <button
                      onClick={() => removeMut.mutate(s.department_id)}
                      disabled={removeMut.isPending}
                      className="p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive rounded-lg transition-colors"
                    >
                      {removeMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
