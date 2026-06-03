import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Edit2, Save, X, CalendarDays } from 'lucide-react';

type LeaveBalance = {
  id: string;
  leave_type_id: string;
  year: parseInt;
  allocated_days: number;
  used_days: number;
  leave_type_name_en: string;
  leave_type_name_ar: string;
};

export const EmployeeLeaveBalances = ({ employeeId }: { employeeId: string }) => {
  const queryClient = useQueryClient();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [allocatedDays, setAllocatedDays] = useState<number>(0);

  const { data: balances, isLoading } = useQuery<LeaveBalance[]>({
    queryKey: ['leave-balances', employeeId],
    queryFn: async () => {
      const res = await api.get(`/leave-balances/employee/${employeeId}?year=${new Date().getFullYear()}`);
      return res.data?.data || [];
    },
    enabled: !!employeeId,
  });

  const updateMutation = useMutation({
    mutationFn: async ({ leaveTypeId, allocated }: { leaveTypeId: string, allocated: number }) => {
      await api.put(`/leave-balances/employee/${employeeId}/${leaveTypeId}`, {
        year: new Date().getFullYear(),
        allocated_days: allocated
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['leave-balances', employeeId] });
      setEditingId(null);
    },
  });

  const handleEdit = (balance: LeaveBalance) => {
    setEditingId(balance.id);
    setAllocatedDays(balance.allocated_days);
  };

  const handleSave = (leaveTypeId: string) => {
    updateMutation.mutate({ leaveTypeId, allocated: allocatedDays });
  };

  if (isLoading) return <Card className="animate-pulse h-32" />;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CalendarDays className="w-5 h-5 text-primary" />
          Leave Balances ({new Date().getFullYear()})
        </CardTitle>
        <CardDescription>Manage this employee's leave allocation for the current year.</CardDescription>
      </CardHeader>
      <CardContent>
        {balances?.length === 0 ? (
          <p className="text-sm text-muted-foreground">No leave balances allocated yet. Sync balances from Leave Config.</p>
        ) : (
          <div className="space-y-4">
            {balances?.map((b) => (
              <div key={b.id} className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 rounded-lg border border-border/50 bg-muted/20">
                <div>
                  <h4 className="font-semibold text-sm">{b.leave_type_name_en}</h4>
                  <p className="text-xs text-muted-foreground">
                    Used: <span className="font-medium text-foreground">{b.used_days}</span> days
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  {editingId === b.id ? (
                    <>
                      <Input 
                        type="number" 
                        value={allocatedDays} 
                        onChange={(e) => setAllocatedDays(Number(e.target.value))}
                        className="w-20 h-8 text-sm"
                      />
                      <Button size="icon" variant="ghost" onClick={() => handleSave(b.leave_type_id)} disabled={updateMutation.isPending} className="h-8 w-8 text-emerald-500">
                        <Save className="w-4 h-4" />
                      </Button>
                      <Button size="icon" variant="ghost" onClick={() => setEditingId(null)} className="h-8 w-8 text-muted-foreground">
                        <X className="w-4 h-4" />
                      </Button>
                    </>
                  ) : (
                    <>
                      <div className="text-right">
                        <p className="text-xs text-muted-foreground mb-1">Allocated</p>
                        <p className="font-semibold text-sm">{b.allocated_days} days</p>
                      </div>
                      <Button size="icon" variant="ghost" onClick={() => handleEdit(b)} className="h-8 w-8 ml-2">
                        <Edit2 className="w-4 h-4" />
                      </Button>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
};
