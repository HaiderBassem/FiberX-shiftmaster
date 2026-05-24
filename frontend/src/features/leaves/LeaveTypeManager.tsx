import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Trash2, Plus, Edit2, CheckCircle2, XCircle } from 'lucide-react';

export const LeaveTypeManager = () => {
  const queryClient = useQueryClient();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState({ name_en: '', name_ar: '', requires_approval: true, color_code: '#3b82f6' });

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
      setFormData({ name_en: '', name_ar: '', requires_approval: true, color_code: '#3b82f6' });
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
      color_code: type.color_code || '#3b82f6',
    });
  };

  const handleAddNew = () => {
    setEditingId('new');
    setFormData({ name_en: '', name_ar: '', requires_approval: true, color_code: '#3b82f6' });
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-foreground">Leave Types Configuration</h2>
          <p className="text-sm text-muted-foreground">Manage the types of leave available to employees.</p>
        </div>
        <Button onClick={handleAddNew} disabled={editingId === 'new'} className="gap-2">
          <Plus className="w-4 h-4" /> Add New
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {editingId === 'new' && (
          <Card className="border-primary">
            <CardHeader>
              <CardTitle className="text-base">New Leave Type</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Name (English)</Label>
                <Input value={formData.name_en} onChange={e => setFormData({ ...formData, name_en: e.target.value })} placeholder="e.g. Annual" />
              </div>
              <div className="space-y-2">
                <Label>Name (Arabic)</Label>
                <Input value={formData.name_ar} onChange={e => setFormData({ ...formData, name_ar: e.target.value })} placeholder="e.g. سنوي" dir="rtl" />
              </div>
              <div className="flex items-center gap-2">
                <input type="checkbox" id="reqAppNew" checked={formData.requires_approval} onChange={e => setFormData({ ...formData, requires_approval: e.target.checked })} />
                <Label htmlFor="reqAppNew">Requires Approval</Label>
              </div>
              <div className="space-y-2">
                <Label>Color Code</Label>
                <div className="flex gap-2 items-center">
                  <input type="color" value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} className="w-8 h-8 rounded border" />
                  <Input value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} />
                </div>
              </div>
            </CardContent>
            <CardFooter className="flex gap-2">
              <Button variant="outline" onClick={() => setEditingId(null)} className="w-full">Cancel</Button>
              <Button onClick={() => saveMutation.mutate(formData)} disabled={!formData.name_en || !formData.name_ar || saveMutation.isPending} className="w-full">Save</Button>
            </CardFooter>
          </Card>
        )}

        {isLoading ? (
          <div className="col-span-full text-center text-muted-foreground py-8">Loading leave types...</div>
        ) : (
          leaveTypes?.map((type: any) => (
            editingId === type.id ? (
              <Card key={type.id} className="border-primary">
                <CardHeader>
                  <CardTitle className="text-base">Edit {type.name_en}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label>Name (English)</Label>
                    <Input value={formData.name_en} onChange={e => setFormData({ ...formData, name_en: e.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label>Name (Arabic)</Label>
                    <Input value={formData.name_ar} onChange={e => setFormData({ ...formData, name_ar: e.target.value })} dir="rtl" />
                  </div>
                  <div className="flex items-center gap-2">
                    <input type="checkbox" id={`reqApp${type.id}`} checked={formData.requires_approval} onChange={e => setFormData({ ...formData, requires_approval: e.target.checked })} />
                    <Label htmlFor={`reqApp${type.id}`}>Requires Approval</Label>
                  </div>
                  <div className="space-y-2">
                    <Label>Color Code</Label>
                    <div className="flex gap-2 items-center">
                      <input type="color" value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} className="w-8 h-8 rounded border" />
                      <Input value={formData.color_code} onChange={e => setFormData({ ...formData, color_code: e.target.value })} />
                    </div>
                  </div>
                </CardContent>
                <CardFooter className="flex gap-2">
                  <Button variant="outline" onClick={() => setEditingId(null)} className="w-full">Cancel</Button>
                  <Button onClick={() => saveMutation.mutate({ ...formData, id: type.id })} disabled={!formData.name_en || !formData.name_ar || saveMutation.isPending} className="w-full">Save</Button>
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
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive" onClick={() => { if(confirm('Delete this leave type?')) deleteMutation.mutate(type.id); }}>
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    {type.requires_approval ? <CheckCircle2 className="w-4 h-4 text-emerald-500" /> : <XCircle className="w-4 h-4 text-amber-500" />}
                    {type.requires_approval ? 'Requires Approval' : 'Auto-approved'}
                  </div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mt-1">
                    {type.is_active ? <CheckCircle2 className="w-4 h-4 text-emerald-500" /> : <XCircle className="w-4 h-4 text-destructive" />}
                    {type.is_active ? 'Active' : 'Inactive'}
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
