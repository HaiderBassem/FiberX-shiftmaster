import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Trash2, Plus, Edit2, CheckCircle2, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export const LeaveTypeManager = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState({ name_en: '', name_ar: '', requires_approval: true, is_active: true, color_code: '#3b82f6', days_per_year: 0, carries_forward: false, unit: 'days', reset_cycle: 'annual' });

  const { data: leaveTypes, isLoading } = useQuery({
    queryKey: ['leave-types'],
    queryFn: async () => {
      const response = await api.get('/leave-types');
      return response.data?.data || [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: async (payload: any) => {
      if (editingId && editingId !== 'new') {
        return await api.put(`/leave-types/${editingId}`, payload);
      } else {
        return await api.post('/leave-types', payload);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['leave-types'] });
      setEditingId(null);
      setFormData({ name_en: '', name_ar: '', requires_approval: true, is_active: true, color_code: '#3b82f6', days_per_year: 0, carries_forward: false, unit: 'days', reset_cycle: 'annual' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/leave-types/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['leave-types'] });
    },
  });

  const handleEdit = (type: any) => {
    setEditingId(type.id);
    setFormData({
      name_en: type.name_en,
      name_ar: type.name_ar,
      requires_approval: type.requires_approval,
      is_active: type.is_active,
      color_code: type.color_code || '#3b82f6',
      days_per_year: type.days_per_year || 0,
      carries_forward: type.carries_forward || false,
      unit: type.unit || 'days',
      reset_cycle: type.reset_cycle || 'annual',
    });
  };

  const handleAddNew = () => {
    setEditingId('new');
    setFormData({ name_en: '', name_ar: '', requires_approval: true, is_active: true, color_code: '#3b82f6', days_per_year: 0, carries_forward: false, unit: 'days', reset_cycle: 'annual' });
  };

  const syncMutation = useMutation({
    mutationFn: async () => {
      await api.post('/leave-balances/sync', { year: new Date().getFullYear() });
    },
    onSuccess: () => {
      alert(t('leaves.sync_success'));
    },
    onError: () => {
      alert(t('leaves.sync_failed'));
    }
  });

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-foreground">{t('leaves.leave_types_configuration')}</h2>
          <p className="text-sm text-muted-foreground">{t('leaves.manage_leave_types')}</p>
        </div>
        <div className="flex items-center gap-3">
          <Button 
            variant="outline" 
            onClick={() => {
              if (confirm(t('leaves.sync_confirm'))) {
                syncMutation.mutate();
              }
            }}
            disabled={syncMutation.isPending}
            className="gap-2 border-primary/20 text-primary hover:bg-primary/10"
          >
            {syncMutation.isPending ? t('leaves.syncing') : t('leaves.sync_balances')}
          </Button>
          <Button onClick={handleAddNew} disabled={editingId === 'new'} className="gap-2">
            <Plus className="w-4 h-4" /> {t('leaves.add_new')}
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {editingId === 'new' && (
          <Card className="border-primary">
            <CardHeader>
              <CardTitle className="text-base">{t('leaves.new_leave_type')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>{t('leaves.name_english')}</Label>
                <Input value={formData.name_en} onChange={e => setFormData({ ...formData, name_en: e.target.value })} placeholder="e.g. Annual" />
              </div>
              <div className="space-y-2">
                <Label>{t('leaves.name_arabic')}</Label>
                <Input value={formData.name_ar} onChange={e => setFormData({ ...formData, name_ar: e.target.value })} placeholder="e.g. سنوي" dir="rtl" />
              </div>
              <div className="flex items-center gap-2">
                <input type="checkbox" id="reqAppNew" checked={formData.requires_approval} onChange={e => setFormData({ ...formData, requires_approval: e.target.checked })} />
                <Label htmlFor="reqAppNew">{t('leaves.requires_approval')}</Label>
              </div>
              <div className="flex items-center gap-2">
                <input type="checkbox" id="isActiveNew" checked={formData.is_active} onChange={e => setFormData({ ...formData, is_active: e.target.checked })} />
                <Label htmlFor="isActiveNew">{t('leaves.is_active')}</Label>
              </div>
              <div className="space-y-2">
                <Label>{t('leaves.quota')} ({formData.unit === 'hours' ? t('leaves.hours') : t('leaves.days')} {t('leaves.per')} {formData.reset_cycle === 'monthly' ? t('leaves.month') : t('leaves.year')})</Label>
                <Input type="number" min="0" value={formData.days_per_year} onChange={e => setFormData({ ...formData, days_per_year: parseInt(e.target.value) || 0 })} placeholder="e.g. 21" />
              </div>
              <div className="flex items-center gap-2">
                <input type="checkbox" id="carriesForwardNew" checked={formData.carries_forward} onChange={e => setFormData({ ...formData, carries_forward: e.target.checked })} />
                <Label htmlFor="carriesForwardNew">{t('leaves.accumulates_carries_forward')}</Label>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-2">
                  <Label>{t('leaves.unit')}</Label>
                  <select 
                    className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                    value={formData.unit} 
                    onChange={e => setFormData({ ...formData, unit: e.target.value })}
                  >
                    <option value="days">{t('leaves.days')}</option>
                    <option value="hours">{t('leaves.hours')}</option>
                  </select>
                </div>
                <div className="space-y-2">
                  <Label>{t('leaves.reset_cycle')}</Label>
                  <select 
                    className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                    value={formData.reset_cycle} 
                    onChange={e => setFormData({ ...formData, reset_cycle: e.target.value })}
                  >
                    <option value="annual">{t('leaves.annual')}</option>
                    <option value="monthly">{t('leaves.monthly')}</option>
                  </select>
                </div>
              </div>
              <div className="space-y-2">
                <Label>{t('leaves.color_code')}</Label>
                <div className="flex gap-2 items-center">
                  <input type="color" value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} className="w-8 h-8 rounded border" />
                  <Input value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} />
                </div>
              </div>
            </CardContent>
            <CardFooter className="flex gap-2">
              <Button variant="outline" onClick={() => setEditingId(null)} className="w-full">{t('common.cancel')}</Button>
              <Button onClick={() => saveMutation.mutate(formData)} disabled={!formData.name_en || !formData.name_ar || saveMutation.isPending} className="w-full">{t('common.save')}</Button>
            </CardFooter>
          </Card>
        )}

        {isLoading ? (
          <div className="col-span-full text-center text-muted-foreground py-8">{t('leaves.loading_types')}</div>
        ) : (
          leaveTypes?.map((type: any) => (
            editingId === type.id ? (
              <Card key={type.id} className="border-primary">
                <CardHeader>
                  <CardTitle className="text-base">{t('leaves.edit')} {type.name_en}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label>{t('leaves.name_english')}</Label>
                    <Input value={formData.name_en} onChange={e => setFormData({ ...formData, name_en: e.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label>{t('leaves.name_arabic')}</Label>
                    <Input value={formData.name_ar} onChange={e => setFormData({ ...formData, name_ar: e.target.value })} dir="rtl" />
                  </div>
                  <div className="flex items-center gap-2">
                    <input type="checkbox" id={`reqApp${type.id}`} checked={formData.requires_approval} onChange={e => setFormData({ ...formData, requires_approval: e.target.checked })} />
                    <Label htmlFor={`reqApp${type.id}`}>{t('leaves.requires_approval')}</Label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input type="checkbox" id={`isActive${type.id}`} checked={formData.is_active} onChange={e => setFormData({ ...formData, is_active: e.target.checked })} />
                    <Label htmlFor={`isActive${type.id}`}>{t('leaves.is_active')}</Label>
                  </div>
                  <div className="space-y-2">
                    <Label>{t('leaves.quota')} ({formData.unit === 'hours' ? t('leaves.hours') : t('leaves.days')} {t('leaves.per')} {formData.reset_cycle === 'monthly' ? t('leaves.month') : t('leaves.year')})</Label>
                    <Input type="number" min="0" value={formData.days_per_year} onChange={e => setFormData({ ...formData, days_per_year: parseInt(e.target.value) || 0 })} />
                  </div>
                  <div className="flex items-center gap-2">
                    <input type="checkbox" id={`carriesForward${type.id}`} checked={formData.carries_forward} onChange={e => setFormData({ ...formData, carries_forward: e.target.checked })} />
                    <Label htmlFor={`carriesForward${type.id}`}>{t('leaves.accumulates_carries_forward')}</Label>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div className="space-y-2">
                      <Label>{t('leaves.unit')}</Label>
                      <select 
                        className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                        value={formData.unit} 
                        onChange={e => setFormData({ ...formData, unit: e.target.value })}
                      >
                        <option value="days">{t('leaves.days')}</option>
                        <option value="hours">{t('leaves.hours')}</option>
                      </select>
                    </div>
                    <div className="space-y-2">
                      <Label>{t('leaves.reset_cycle')}</Label>
                      <select 
                        className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                        value={formData.reset_cycle} 
                        onChange={e => setFormData({ ...formData, reset_cycle: e.target.value })}
                      >
                        <option value="annual">{t('leaves.annual')}</option>
                        <option value="monthly">{t('leaves.monthly')}</option>
                      </select>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>{t('leaves.color_code')}</Label>
                    <div className="flex gap-2 items-center">
                      <input type="color" value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} className="w-8 h-8 rounded border" />
                      <Input value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} />
                    </div>
                  </div>
                </CardContent>
                <CardFooter className="flex gap-2">
                  <Button variant="outline" onClick={() => setEditingId(null)} className="w-full">{t('common.cancel')}</Button>
                  <Button onClick={() => saveMutation.mutate({ ...formData, id: type.id })} disabled={!formData.name_en || !formData.name_ar || saveMutation.isPending} className="w-full">{t('common.save')}</Button>
                </CardFooter>
              </Card>
            ) : (
              <Card key={type.id}>
                <CardHeader className="pb-2 flex flex-row items-start justify-between">
                  <div>
                    <CardTitle className="text-lg flex items-center gap-2">
                      <span className="w-3 h-3 rounded-full" style={{ backgroundColor: type.color_code || '#cbd5e1' }} />
                      {type.name_en}
                    </CardTitle>
                    <p className="text-sm text-muted-foreground mt-1" dir="rtl">{type.name_ar}</p>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-primary" onClick={() => handleEdit(type)}>
                      <Edit2 className="w-4 h-4" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive" onClick={() => { if(confirm(t('leaves.delete_confirm'))) deleteMutation.mutate(type.id); }}>
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    {type.requires_approval ? <CheckCircle2 className="w-4 h-4 text-emerald-500" /> : <XCircle className="w-4 h-4 text-amber-500" />}
                    {type.requires_approval ? t('leaves.requires_approval') : t('leaves.auto_approved')}
                  </div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mt-1">
                    {type.is_active ? <CheckCircle2 className="w-4 h-4 text-emerald-500" /> : <XCircle className="w-4 h-4 text-destructive" />}
                    {type.is_active ? t('leaves.active') : t('leaves.inactive')}
                  </div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mt-1">
                    <span className="font-medium text-foreground">{t('leaves.quota')}:</span> {type.days_per_year} {type.unit === 'hours' ? t('leaves.hours') : t('leaves.days')}/{type.reset_cycle === 'monthly' ? t('leaves.month') : t('leaves.year')}
                    {type.carries_forward && <span className="text-xs bg-muted px-1.5 py-0.5 rounded ml-2">{t('leaves.accumulates')}</span>}
                  </div>
                </CardContent>
              </Card>
            )
          ))
        )}
      </div>
    </div>
  );
};
