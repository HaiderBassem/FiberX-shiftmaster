import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import api from '@/lib/api';
import type { Province } from './types';
import { X, Loader2, Plus, Trash2, Edit2, Check, X as XIcon } from 'lucide-react';

export function ProvinceManager({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [newName, setNewName] = useState('');
  const [isAdding, setIsAdding] = useState(false);
  const [error, setError] = useState('');

  const { data: provinces, isLoading } = useQuery<Province[]>({
    queryKey: ['provinces'],
    queryFn: async () => (await api.get('/provinces')).data.data ?? [],
  });

  const createMut = useMutation({
    mutationFn: (name: string) => api.post('/provinces', { name, sort_order: 0, is_active: true }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['provinces'] });
      setIsAdding(false);
      setNewName('');
      setError('');
    },
    onError: (err: any) => setError(err.response?.data?.error || t('services.error_occurred'))
  });

  const updateMut = useMutation({
    mutationFn: ({ id, name, sort_order, is_active }: { id: string, name: string, sort_order: number, is_active: boolean }) =>
      api.put(`/provinces/${id}`, { name, sort_order, is_active }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['provinces'] });
      setEditingId(null);
      setError('');
    },
    onError: (err: any) => setError(err.response?.data?.error || t('services.error_occurred'))
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/provinces/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['provinces'] });
    },
    onError: (err: any) => setError(err.response?.data?.error || t('services.error_occurred'))
  });

  const handleCreate = () => {
    if (!newName.trim()) return;
    createMut.mutate(newName.trim());
  };

  const handleUpdate = (p: Province) => {
    if (!editName.trim()) return;
    updateMut.mutate({ id: p.id, name: editName.trim(), sort_order: p.sort_order, is_active: p.is_active });
  };

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-card border border-border rounded-2xl w-full max-w-lg shadow-2xl flex flex-col max-h-[90vh]">
        <div className="flex items-center justify-between px-6 py-5 border-b border-border">
          <h2 className="text-lg font-bold text-foreground">
            {t('services.manage_provinces')}
          </h2>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>

        <div className="p-6 overflow-y-auto flex-1 space-y-4">
          {error && (
            <div className="p-3 bg-destructive/10 border border-destructive/20 text-destructive rounded-xl text-sm">
              {error}
            </div>
          )}

          <div className="flex justify-between items-center mb-4">
            <p className="text-sm text-muted-foreground">{t('services.provinces_list')}</p>
            {!isAdding && (
              <button
                onClick={() => setIsAdding(true)}
                className="flex items-center gap-1.5 text-xs font-medium text-primary hover:text-primary/80 transition-colors bg-primary/10 px-3 py-1.5 rounded-lg"
              >
                <Plus className="w-3.5 h-3.5" />
                {t('services.add_province')}
              </button>
            )}
          </div>

          {isAdding && (
            <div className="flex items-center gap-2 mb-4 bg-muted/30 p-2 rounded-xl border border-border">
              <input
                autoFocus
                value={newName}
                onChange={e => setNewName(e.target.value)}
                placeholder={t('services.province_name')}
                className="flex-1 bg-background border border-border rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-primary/50"
                onKeyDown={e => e.key === 'Enter' && handleCreate()}
              />
              <button
                onClick={handleCreate}
                disabled={createMut.isPending}
                className="p-1.5 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50"
              >
                {createMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
              </button>
              <button
                onClick={() => { setIsAdding(false); setNewName(''); setError(''); }}
                className="p-1.5 bg-muted text-muted-foreground rounded-lg hover:bg-muted/80"
              >
                <XIcon className="w-4 h-4" />
              </button>
            </div>
          )}

          {isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="w-6 h-6 animate-spin text-primary" />
            </div>
          ) : (
            <div className="space-y-2">
              {provinces?.map(p => (
                <div key={p.id} className="flex items-center justify-between p-3 bg-background border border-border rounded-xl">
                  {editingId === p.id ? (
                    <div className="flex items-center gap-2 flex-1 mr-2">
                      <input
                        autoFocus
                        value={editName}
                        onChange={e => setEditName(e.target.value)}
                        className="flex-1 bg-card border border-border rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-primary/50"
                        onKeyDown={e => e.key === 'Enter' && handleUpdate(p)}
                      />
                      <button
                        onClick={() => handleUpdate(p)}
                        disabled={updateMut.isPending}
                        className="p-1.5 text-emerald-500 hover:bg-emerald-500/10 rounded-lg transition-colors"
                      >
                        {updateMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
                      </button>
                      <button
                        onClick={() => setEditingId(null)}
                        className="p-1.5 text-muted-foreground hover:bg-muted rounded-lg transition-colors"
                      >
                        <XIcon className="w-4 h-4" />
                      </button>
                    </div>
                  ) : (
                    <>
                      <span className="text-sm font-medium">{p.name}</span>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => { setEditingId(p.id); setEditName(p.name); setError(''); }}
                          className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg transition-colors"
                        >
                          <Edit2 className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => {
                            if (window.confirm(t('services.confirm_delete_province'))) {
                              deleteMut.mutate(p.id);
                            }
                          }}
                          disabled={deleteMut.isPending}
                          className="p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </>
                  )}
                </div>
              ))}
              {provinces?.length === 0 && !isAdding && (
                <p className="text-center text-sm text-muted-foreground py-4">{t('services.no_provinces')}</p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
