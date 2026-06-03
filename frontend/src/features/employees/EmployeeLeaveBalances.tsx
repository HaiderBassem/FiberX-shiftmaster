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
  allocated_amount: number;
  used_amount: number;
  leave_type_name_en: string;
  leave_type_name_ar: string;
  month: number;
  unit: string;
  reset_cycle: string;
};

export const EmployeeLeaveBalances = ({ employeeId }: { employeeId: string }) => {
  const queryClient = useQueryClient();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [allocatedAmount, setAllocatedAmount] = useState<number>(0);

  const { data: balances, isLoading } = useQuery<LeaveBalance[]>({
    queryKey: ['leave-balances', employeeId],
    queryFn: async () => {
      const res = await api.get(`/leave-balances/employee/${employeeId}?year=${new Date().getFullYear()}`);
      return res.data?.data || [];
    },
    enabled: !!employeeId,
  });

  const updateMutation = useMutation({
    mutationFn: async ({ leaveTypeId, month, allocated }: { leaveTypeId: string, month: number, allocated: number }) => {
      await api.put(`/leave-balances/employee/${employeeId}/${leaveTypeId}`, {
        year: new Date().getFullYear(),
        month: month,
        allocated_amount: allocated
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['leave-balances', employeeId] });
      setEditingId(null);
    },
  });

  const handleEdit = (balance: LeaveBalance) => {
    setEditingId(balance.id);
    setAllocatedAmount(balance.allocated_amount);
  };

  const handleSave = (leaveTypeId: string, month: number) => {
    updateMutation.mutate({ leaveTypeId, month, allocated: allocatedAmount });
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
          <div className="space-y-6">
            {Object.values(
              balances?.reduce((acc: any, b: any) => {
                if (!acc[b.leave_type_id]) {
                  acc[b.leave_type_id] = {
                    name: b.leave_type_name_en,
                    unit: b.unit,
                    reset_cycle: b.reset_cycle,
                    balances: []
                  };
                }
                acc[b.leave_type_id].balances.push(b);
                return acc;
              }, {}) || {}
            ).map((group: any) => (
              <div key={group.name} className="p-4 rounded-xl border border-border/50 bg-card shadow-sm">
                <div className="flex items-center justify-between mb-4 border-b border-border/50 pb-3">
                  <h3 className="font-semibold text-lg text-foreground">{group.name}</h3>
                  <span className="text-xs font-semibold px-2.5 py-1 bg-primary/10 text-primary rounded-full uppercase tracking-wider">
                    {group.reset_cycle} • {group.unit}
                  </span>
                </div>
                
                {group.reset_cycle === 'annual' ? (
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 rounded-lg bg-muted/30">
                    {group.balances.map((b: any) => (
                      <div key={b.id} className="flex flex-col sm:flex-row sm:items-center justify-between w-full gap-3">
                        <div>
                          <p className="text-sm text-muted-foreground">
                            Used: <span className="font-medium text-foreground">{b.used_amount}</span> {b.unit}
                          </p>
                        </div>
                        <div className="flex items-center gap-3">
                          {editingId === b.id ? (
                            <>
                              <Input 
                                type="number" 
                                value={allocatedAmount} 
                                onChange={(e) => setAllocatedAmount(Number(e.target.value))}
                                className="w-20 h-8 text-sm"
                              />
                              <Button size="icon" variant="ghost" onClick={() => handleSave(b.leave_type_id, b.month)} disabled={updateMutation.isPending} className="h-8 w-8 text-emerald-500">
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
                                <p className="font-semibold text-sm">{b.allocated_amount} {b.unit}</p>
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
                ) : (
                  <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-3">
                    {group.balances.sort((a: any, b: any) => a.month - b.month).map((b: any) => (
                      <div key={b.id} className="p-3 rounded-lg bg-muted/30 border border-border/40 flex flex-col gap-2">
                        <div className="flex justify-between items-center mb-1">
                          <h4 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Month {b.month}</h4>
                          {editingId !== b.id && (
                            <Button size="icon" variant="ghost" onClick={() => handleEdit(b)} className="h-6 w-6">
                              <Edit2 className="w-3 h-3 text-muted-foreground" />
                            </Button>
                          )}
                        </div>
                        
                        {editingId === b.id ? (
                          <div className="flex items-center gap-2">
                            <Input 
                              type="number" 
                              value={allocatedAmount} 
                              onChange={(e) => setAllocatedAmount(Number(e.target.value))}
                              className="h-8 text-sm"
                            />
                            <Button size="icon" variant="ghost" onClick={() => handleSave(b.leave_type_id, b.month)} disabled={updateMutation.isPending} className="h-8 w-8 text-emerald-500 shrink-0">
                              <Save className="w-4 h-4" />
                            </Button>
                            <Button size="icon" variant="ghost" onClick={() => setEditingId(null)} className="h-8 w-8 text-muted-foreground shrink-0">
                              <X className="w-4 h-4" />
                            </Button>
                          </div>
                        ) : (
                          <div className="flex justify-between items-end mt-1">
                            <div>
                              <p className="text-[10px] text-muted-foreground uppercase mb-0.5">Used</p>
                              <p className="text-sm font-medium">{b.used_amount}</p>
                            </div>
                            <div className="text-right">
                              <p className="text-[10px] text-muted-foreground uppercase mb-0.5">Allocated</p>
                              <p className="text-sm font-semibold">{b.allocated_amount}</p>
                            </div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
};
