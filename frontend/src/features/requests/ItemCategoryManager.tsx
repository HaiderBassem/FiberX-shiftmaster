import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { X, Plus, Edit2, Trash2, Save } from 'lucide-react';
import { Button } from '../../components/ui/button';
import api from '../../lib/api';
import { useTranslation } from 'react-i18next';

interface Category {
  id: string;
  name: string;
  to_emails: string;
  cc_emails: string | null;
}

interface ItemCategoryManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ItemCategoryManager = ({ isOpen, onClose }: ItemCategoryManagerProps) => {
  const { t } = useTranslation();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState({ name: '', to_emails: '', cc_emails: '' });

  const { data: categories, isLoading, refetch } = useQuery({
    queryKey: ['item-categories-manage'],
    queryFn: async () => {
      const { data } = await api.get('/item-requests/categories');
      return data.data as Category[];
    },
    enabled: isOpen,
  });

  const saveMutation = useMutation({
    mutationFn: async (data: any) => {
      if (data.id) {
        await api.put(`/item-requests/categories/${data.id}`, data);
      } else {
        await api.post('/item-requests/categories', data);
      }
    },
    onSuccess: () => {
      refetch();
      setEditingId(null);
      setFormData({ name: '', to_emails: '', cc_emails: '' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/item-requests/categories/${id}`);
    },
    onSuccess: () => refetch(),
  });

  const handleEdit = (cat: Category) => {
    setEditingId(cat.id);
    setFormData({
      name: cat.name,
      to_emails: cat.to_emails,
      cc_emails: cat.cc_emails || '',
    });
  };

  const handleSave = () => {
    if (!formData.name.trim() || !formData.to_emails.trim()) return;
    saveMutation.mutate({
      id: editingId === 'new' ? undefined : editingId,
      name: formData.name,
      to_emails: formData.to_emails,
      cc_emails: formData.cc_emails.trim() ? formData.cc_emails : null,
    });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
      <div className="bg-card w-full max-w-2xl max-h-[85vh] flex flex-col rounded-2xl shadow-xl border border-border animate-in zoom-in-95 duration-200">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold text-foreground">{t('items.manage_item_categories')}</h2>
          <button onClick={onClose} className="p-2 hover:bg-muted rounded-full transition-colors text-muted-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>
        
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          <div className="flex justify-end">
            <Button 
              size="sm" 
              onClick={() => {
                setEditingId('new');
                setFormData({ name: '', to_emails: '', cc_emails: '' });
              }}
              disabled={editingId === 'new'}
            >
              <Plus className="w-4 h-4 mr-2" />
              {t('items.add_category')}
            </Button>
          </div>

          <div className="space-y-3">
            {isLoading ? (
              <div className="py-8 text-center text-muted-foreground">Loading...</div>
            ) : categories?.length === 0 && editingId !== 'new' ? (
              <div className="py-8 text-center text-muted-foreground border-2 border-dashed border-border rounded-xl">
                {t('items.no_categories_configured')}
              </div>
            ) : (
              <>
                {editingId === 'new' && (
                  <div className="p-4 bg-muted/30 rounded-xl border border-border space-y-3">
                    <div>
                      <label className="text-xs font-semibold text-muted-foreground mb-1 block">{t('items.category_name')}</label>
                      <input 
                        type="text" 
                        value={formData.name}
                        onChange={e => setFormData({...formData, name: e.target.value})}
                        className="w-full px-3 py-1.5 bg-background border border-input rounded-lg text-sm"
                        placeholder="e.g. IT Equipment"
                      />
                    </div>
                    <div>
                      <label className="text-xs font-semibold text-muted-foreground mb-1 block">{t('items.to_emails')}</label>
                      <input 
                        type="text" 
                        value={formData.to_emails}
                        onChange={e => setFormData({...formData, to_emails: e.target.value})}
                        className="w-full px-3 py-1.5 bg-background border border-input rounded-lg text-sm"
                        placeholder="it@company.com"
                      />
                    </div>
                    <div>
                      <label className="text-xs font-semibold text-muted-foreground mb-1 block">{t('items.cc_emails')}</label>
                      <input 
                        type="text" 
                        value={formData.cc_emails}
                        onChange={e => setFormData({...formData, cc_emails: e.target.value})}
                        className="w-full px-3 py-1.5 bg-background border border-input rounded-lg text-sm"
                        placeholder="manager@company.com"
                      />
                    </div>
                    <div className="flex justify-end gap-2 pt-2">
                      <Button size="sm" variant="outline" onClick={() => setEditingId(null)}>{t('common.cancel')}</Button>
                      <Button size="sm" onClick={handleSave} disabled={saveMutation.isPending}>
                        <Save className="w-4 h-4 mr-2" />
                        {t('common.save')}
                      </Button>
                    </div>
                  </div>
                )}
                
                {categories?.map((cat) => (
                  <div key={cat.id} className="p-4 bg-background rounded-xl border border-border flex flex-col sm:flex-row gap-4 justify-between sm:items-start transition-all hover:border-primary/30 shadow-sm">
                    {editingId === cat.id ? (
                      <div className="flex-1 space-y-3">
                        <div>
                          <label className="text-xs font-semibold text-muted-foreground mb-1 block">{t('items.category_name')}</label>
                          <input 
                            type="text" 
                            value={formData.name}
                            onChange={e => setFormData({...formData, name: e.target.value})}
                            className="w-full px-3 py-1.5 bg-background border border-input rounded-lg text-sm"
                          />
                        </div>
                        <div>
                          <label className="text-xs font-semibold text-muted-foreground mb-1 block">{t('items.to_emails')}</label>
                          <input 
                            type="text" 
                            value={formData.to_emails}
                            onChange={e => setFormData({...formData, to_emails: e.target.value})}
                            className="w-full px-3 py-1.5 bg-background border border-input rounded-lg text-sm"
                          />
                        </div>
                        <div>
                          <label className="text-xs font-semibold text-muted-foreground mb-1 block">{t('items.cc_emails')}</label>
                          <input 
                            type="text" 
                            value={formData.cc_emails}
                            onChange={e => setFormData({...formData, cc_emails: e.target.value})}
                            className="w-full px-3 py-1.5 bg-background border border-input rounded-lg text-sm"
                          />
                        </div>
                        <div className="flex gap-2 pt-2">
                          <Button size="sm" onClick={handleSave} disabled={saveMutation.isPending}>{t('common.save')}</Button>
                          <Button size="sm" variant="outline" onClick={() => setEditingId(null)}>{t('common.cancel')}</Button>
                        </div>
                      </div>
                    ) : (
                      <>
                        <div className="flex-1">
                          <h4 className="font-semibold text-foreground">{cat.name}</h4>
                          <div className="mt-2 space-y-1 text-sm text-muted-foreground">
                            <p><span className="font-medium">To:</span> {cat.to_emails}</p>
                            {cat.cc_emails && <p><span className="font-medium">CC:</span> {cat.cc_emails}</p>}
                          </div>
                        </div>
                        <div className="flex items-center gap-1">
                          <Button size="icon" variant="ghost" onClick={() => handleEdit(cat)}>
                            <Edit2 className="w-4 h-4 text-muted-foreground" />
                          </Button>
                          <Button 
                            size="icon" 
                            variant="ghost" 
                            className="text-destructive hover:text-destructive hover:bg-destructive/10"
                            onClick={() => {
                              if (confirm(t('items.delete_category_confirm'))) {
                                deleteMutation.mutate(cat.id);
                              }
                            }}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </>
                    )}
                  </div>
                ))}
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
