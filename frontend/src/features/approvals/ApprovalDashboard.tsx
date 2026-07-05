import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ShieldCheck, CalendarOff, ArrowLeftRight, Users, AlertTriangle, CheckCircle2, XCircle,
  UserPlus, History, Clock, ChevronDown, ChevronUp, Moon, Info, Package } from 'lucide-react';
import { format } from 'date-fns';
import { fmtDate, fmtDateTime, parseDate } from '@/lib/dateUtils';
import { useTranslation } from 'react-i18next';

// ─── Coverage Preview Widget ───────────────────────────────────────────────────
const CoveragePreview = ({ shiftId, date }: { shiftId: string; date: string }) => {
  const { t } = useTranslation();
  const { data: coverage, isLoading } = useQuery({
    queryKey: ['coverage', shiftId, date],
    queryFn: async () => { const res = await api.get(`/leaves/coverage-preview?shift_id=${shiftId}&date=${date}`); return res.data?.data; },
    enabled: !!shiftId && !!date,
  });

  if (isLoading) return <div className="animate-pulse h-16 bg-muted/50 rounded-lg" />;
  if (!coverage) return null;

  const isLow = coverage.total_working <= 2;

  return (
    <div className={`p-3 rounded-lg border ${isLow ? 'bg-destructive/5 border-destructive/30' : 'bg-emerald-500/5 border-emerald-500/20'}`}>
      <div className="flex items-center gap-2 mb-2">
        {isLow ? <AlertTriangle className="w-4 h-4 text-destructive" /> : <Users className="w-4 h-4 text-emerald-500" />}
        <span className={`text-xs font-semibold uppercase tracking-wider ${isLow ? 'text-destructive' : 'text-emerald-500'}`}>
          {t('approvals.shift_coverage')} — {date}
        </span>
      </div>
      <div className="grid grid-cols-4 gap-2 text-center">
        <div><div className="text-lg font-bold text-foreground">{coverage.total_assigned}</div><div className="text-[10px] text-muted-foreground uppercase">{t('approvals.total')}</div></div>
        <div><div className="text-lg font-bold text-emerald-500">{coverage.total_working}</div><div className="text-[10px] text-muted-foreground uppercase">{t('approvals.working')}</div></div>
        <div><div className="text-lg font-bold text-amber-500">{coverage.total_off}</div><div className="text-[10px] text-muted-foreground uppercase">{t('approvals.off')}</div></div>
        <div><div className="text-lg font-bold text-destructive">{coverage.total_on_leave}</div><div className="text-[10px] text-muted-foreground uppercase">{t('approvals.on_leave')}</div></div>
      </div>
      {isLow && (
        <p className="text-xs text-destructive mt-2 flex items-center gap-1">
          <AlertTriangle className="w-3 h-3" /> {t('approvals.warning_low_staffing')}
        </p>
      )}
    </div>
  );
};

// ─── Night Shift Previous Day Info ─────────────────────────────────────────────
const NightShiftPrevDayInfo = ({ shiftId, date }: { shiftId: string; date: string }) => {
  const { t } = useTranslation();
  // Check coverage for the day BEFORE the leave
  const prevDate = (() => {
    const d = parseDate(date);
    d.setDate(d.getDate() - 1);
    return format(d, 'yyyy-MM-dd');
  })();

  const { data: coverage, isLoading } = useQuery({
    queryKey: ['coverage', shiftId, prevDate],
    queryFn: async () => { const res = await api.get(`/leaves/coverage-preview?shift_id=${shiftId}&date=${prevDate}`); return res.data?.data; },
    enabled: !!shiftId && !!prevDate,
  });

  if (isLoading) return <div className="animate-pulse h-12 bg-muted/50 rounded-lg" />;
  if (!coverage) return null;

  return (
    <div className="p-3 rounded-lg border bg-indigo-500/5 border-indigo-500/20">
      <div className="flex items-center gap-2 mb-2">
        <Moon className="w-4 h-4 text-indigo-400" />
        <span className="text-xs font-semibold uppercase tracking-wider text-indigo-400">
          {t('approvals.night_shift_prev_day')} ({prevDate})
        </span>
      </div>
      <div className="grid grid-cols-3 gap-2 text-center">
        <div><div className="text-lg font-bold text-amber-500">{coverage.total_off}</div><div className="text-[10px] text-muted-foreground uppercase">{t('approvals.off')}</div></div>
        <div><div className="text-lg font-bold text-destructive">{coverage.total_on_leave}</div><div className="text-[10px] text-muted-foreground uppercase">{t('approvals.on_leave')}</div></div>
        <div><div className="text-lg font-bold text-emerald-500">{coverage.total_working}</div><div className="text-[10px] text-muted-foreground uppercase">{t('approvals.working')}</div></div>
      </div>
      {(coverage.total_off + coverage.total_on_leave) > 0 && (
        <p className="text-xs text-indigo-400 mt-2 flex items-center gap-1">
          <Info className="w-3 h-3" /> {coverage.total_off + coverage.total_on_leave} {t('approvals.employees_were_off')}
        </p>
      )}
    </div>
  );
};

// ─── Quick Replacement Select ──────────────────────────────────────────────────
const QuickReplacementSelect = ({ date, onSelect }: { date: string; onSelect: (empId: string) => void }) => {
  const { t } = useTranslation();
  const { data: replacements, isLoading } = useQuery({
    queryKey: ['replacements', date],
    queryFn: async () => { const res = await api.get(`/schedules/replacements?date=${date}`); return res.data?.data || []; },
    enabled: !!date,
  });

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  return (
    <div className="space-y-2">
      <Label className="flex items-center gap-1"><UserPlus className="w-4 h-4" /> {t('approvals.quick_replacement')}</Label>
      <select className={selectClass} onChange={(e) => onSelect(e.target.value)} disabled={isLoading}>
        <option value="">{t('approvals.select_replacement')}</option>
        {replacements?.map((emp: any) => (
          <option key={emp.id} value={emp.id}>{emp.first_name} {emp.last_name} — {emp.employee_code}</option>
        ))}
      </select>
      {replacements?.length === 0 && !isLoading && <p className="text-xs text-muted-foreground">{t('approvals.no_replacements')}</p>}
    </div>
  );
};

// ─── Leave History Tab ─────────────────────────────────────────────────────────
const LeaveHistory = () => {
  const { t } = useTranslation();
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const { user, adminSelectedDepartmentId, managerSelectedDepartmentId } = useAuthStore();
  const queryClient = useQueryClient();
  const selectedDeptId = user?.role === 'admin' ? adminSelectedDepartmentId : (user?.role === 'manager' ? managerSelectedDepartmentId : null);

  const { data: history, isLoading } = useQuery({
    queryKey: ['leaves', 'history', selectedDeptId],
    queryFn: async () => { const res = await api.get('/leaves/history'); return res.data?.data || []; },
  });

  const statusBadge = (status: string) => {
    const map: Record<string, string> = {
      pending: 'bg-amber-500/10 text-amber-500 border-amber-500/20',
      approved_by_team_leader: 'bg-blue-500/10 text-blue-500 border-blue-500/20',
      approved_by_manager: 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20',
      rejected: 'bg-destructive/10 text-destructive border-destructive/20',
      cancelled: 'bg-gray-500/10 text-gray-500 border-gray-500/20',
    };
    const labels: Record<string, string> = {
      pending: t('approvals.pending'),
      approved_by_team_leader: t('approvals.tl_approved'),
      approved_by_manager: t('approvals.fully_approved'),
      rejected: t('approvals.rejected'),
      cancelled: t('approvals.cancelled'),
    };
    return (
      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider border ${map[status] || 'bg-muted text-muted-foreground'}`}>
        {labels[status] || status}
      </span>
    );
  };

  const cancelLeave = useMutation({
    mutationFn: async (leaveId: string) => { await api.post(`/leaves/${leaveId}/cancel-approval`); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['leaves'] }); },
  });

  if (isLoading) return <div className="space-y-3">{[1,2,3].map(i => <Card key={i} className="animate-pulse h-24" />)}</div>;

  return (
    <div className="space-y-3">
      {history?.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center p-10 text-muted-foreground">
            <History className="w-10 h-10 mb-3 opacity-20" />
            <p>{t('approvals.no_leave_history')}</p>
          </CardContent>
        </Card>
      ) : (
        history?.map((item: any) => (
          <Card key={item.leave_id} className="overflow-hidden">
            <CardHeader className="pb-2 cursor-pointer" onClick={() => setExpandedId(expandedId === item.leave_id ? null : item.leave_id)}>
              <div className="flex items-start sm:items-center justify-between gap-2 flex-col sm:flex-row">
                <div>
                  <CardTitle className="text-base flex items-center gap-2 flex-wrap">
                    {item.employee_profile_image && (
                      <img src={item.employee_profile_image} alt="" className="w-5 h-5 rounded-full object-cover" />
                    )}
                    {item.employee_name}
                    <span className="text-xs text-muted-foreground font-mono">({item.employee_code})</span>
                  </CardTitle>
                  <CardDescription className="mt-1">
                    {item.leave_type_name_en?.charAt(0).toUpperCase() + item.leave_type_name_en?.slice(1) || 'Leave'} ·{' '}
                    {fmtDate(item.start_date, 'MMM d')} → {fmtDate(item.end_date, 'MMM d, yyyy')} ·{' '}
                    {item.total_days} day{item.total_days > 1 ? 's' : ''}
                  </CardDescription>
                </div>
                <div className="flex items-center gap-2">
                  {statusBadge(item.status)}
                  {expandedId === item.leave_id ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
                </div>
              </div>
            </CardHeader>
            {expandedId === item.leave_id && (
              <CardContent className="pt-0 space-y-3">
                {item.reason && (
                  <div className="p-3 rounded-lg bg-muted/30 border border-border">
                    <p className="text-xs text-muted-foreground uppercase tracking-wider mb-1">{t('approvals.reason')}</p>
                    <p className="text-sm text-foreground">{item.reason}</p>
                  </div>
                )}
                {item.rejection_reason && (
                  <div className="p-3 rounded-lg bg-destructive/5 border border-destructive/20">
                    <p className="text-xs text-destructive uppercase tracking-wider mb-1">{t('approvals.rejection_reason')}</p>
                    <p className="text-sm text-destructive">{item.rejection_reason}</p>
                  </div>
                )}
                {item.applied_date && (
                  <p className="text-xs text-muted-foreground">{t('approvals.applied')}: {fmtDate(item.applied_date)}</p>
                )}
                <div className="border-t border-border pt-3">
                  <p className="text-xs text-muted-foreground uppercase tracking-wider mb-2 flex items-center gap-1">
                    <Clock className="w-3 h-3" /> {t('approvals.approval_timeline')}
                  </p>
                  {item.approvals?.length > 0 ? (
                    <div className="space-y-2">
                      {item.approvals.map((ap: any, idx: number) => (
                        <div key={idx} className={`flex items-start gap-3 p-2 rounded-lg text-sm ${ap.action === 'approved' ? 'bg-emerald-500/5' : 'bg-destructive/5'}`}>
                          <div className={`mt-0.5 w-5 h-5 rounded-full flex items-center justify-center flex-shrink-0 ${ap.action === 'approved' ? 'bg-emerald-500/20' : 'bg-destructive/20'}`}>
                            {ap.action === 'approved' ? <CheckCircle2 className="w-3 h-3 text-emerald-500" /> : <XCircle className="w-3 h-3 text-destructive" />}
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="font-medium text-foreground">
                              {ap.approver_name}
                              <span className="ml-1 text-xs text-muted-foreground">
                                ({ap.approver_role === 'team_leader' ? 'Team Leader' : ap.approver_role === 'manager' ? 'Manager' : ap.approver_role})
                              </span>
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {ap.action === 'approved' ? t('approvals.approved') : t('approvals.rejected')} · {fmtDateTime(ap.created_at, 'MMM d, yyyy · h:mm a')}
                            </p>
                            {ap.notes && <p className="text-xs text-muted-foreground italic mt-1">"{ap.notes}"</p>}
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-muted-foreground italic">{t('approvals.no_approval_actions')}</p>
                  )}
                  {/* Cancel Approval Button */}
                  {(item.status === 'approved_by_manager' || item.status === 'approved_by_team_leader' || item.status === 'approved') && (
                    <div className="mt-4 flex justify-end">
                      <Button
                        variant="outline"
                        size="sm"
                        className="text-destructive border-destructive/30 hover:bg-destructive/10 gap-1"
                        onClick={() => {
                          if (confirm(t('approvals.confirm_cancel_leave'))) {
                            cancelLeave.mutate(item.leave_id);
                          }
                        }}
                        disabled={cancelLeave.isPending}
                      >
                        <XCircle className="w-4 h-4" /> {t('approvals.cancel_approval')}
                      </Button>
                    </div>
                  )}
                </div>
              </CardContent>
            )}
          </Card>
        ))
      )}
    </div>
  );
};

// ─── Swap History Tab ──────────────────────────────────────────────────────────
const SwapHistory = () => {
  const { t } = useTranslation();
  const { user, adminSelectedDepartmentId, managerSelectedDepartmentId } = useAuthStore();
  const selectedDeptId = user?.role === 'admin' ? adminSelectedDepartmentId : (user?.role === 'manager' ? managerSelectedDepartmentId : null);
  const queryClient = useQueryClient();

  const { data: history, isLoading } = useQuery({
    queryKey: ['swaps', 'history', selectedDeptId],
    queryFn: async () => { const res = await api.get('/swaps/history'); return res.data?.data || []; },
  });

  const cancelSwap = useMutation({
    mutationFn: async (swapId: string) => { await api.post(`/swaps/${swapId}/cancel-approval`); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['swaps'] }); },
  });

  const statusBadge = (status: string) => {
    const map: Record<string, string> = {
      pending: 'bg-amber-500/10 text-amber-500 border-amber-500/20',
      employee_accepted: 'bg-blue-500/10 text-blue-500 border-blue-500/20',
      approved: 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20',
      rejected: 'bg-destructive/10 text-destructive border-destructive/20',
      cancelled: 'bg-gray-500/10 text-gray-500 border-gray-500/20',
    };
    const labels: Record<string, string> = {
      pending: t('approvals.pending'),
      employee_accepted: t('approvals.emp_accepted'),
      approved: t('approvals.approved'),
      rejected: t('approvals.rejected'),
      cancelled: t('approvals.cancelled'),
    };
    return (
      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider border ${map[status] || 'bg-muted text-muted-foreground'}`}>
        {labels[status] || status}
      </span>
    );
  };

  if (isLoading) return <div className="space-y-3">{[1,2].map(i => <Card key={i} className="animate-pulse h-24" />)}</div>;

  return (
    <div className="space-y-3">
      {history?.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center p-10 text-muted-foreground">
            <ArrowLeftRight className="w-10 h-10 mb-3 opacity-20" />
            <p>{t('approvals.no_swap_history')}</p>
          </CardContent>
        </Card>
      ) : (
        history?.map((swap: any) => (
          <Card key={swap.id}>
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <CardTitle className="text-base flex items-center gap-2">
                    {t('approvals.shift_swap')} · {fmtDate(swap.shift_date)}
                  </CardTitle>
                  <CardDescription className="mt-1 flex items-center gap-2 flex-wrap">
                    <span className="flex items-center gap-1">
                      {t('approvals.requester')}
                      {swap.requester_profile_image && <img src={swap.requester_profile_image} alt="" className="w-4 h-4 rounded-full object-cover" />}
                      {swap.requester_name}
                    </span>
                    ↔
                    <span className="flex items-center gap-1">
                      {t('approvals.target')}
                      {swap.target_profile_image && <img src={swap.target_profile_image} alt="" className="w-4 h-4 rounded-full object-cover" />}
                      {swap.target_employee_name}
                    </span>
                  </CardDescription>
                </div>
                {statusBadge(swap.status)}
              </div>
            </CardHeader>
            <CardContent>
              {swap.reason && <p className="text-sm text-muted-foreground italic mb-2">"{swap.reason}"</p>}
              <div className="text-xs text-muted-foreground mt-2">
                {t('approvals.created')}: {fmtDate(swap.created_at)}
                {swap.approval_date && ` · ${t('approvals.approved')}: ${fmtDate(swap.approval_date)}`}
              </div>

              {swap.status === 'approved' && (
                <div className="mt-4 flex justify-end">
                  <Button
                    variant="outline"
                    size="sm"
                    className="text-destructive border-destructive/30 hover:bg-destructive/10 gap-1"
                    onClick={() => {
                      if (confirm(t('approvals.confirm_cancel_swap'))) {
                        cancelSwap.mutate(swap.id);
                      }
                    }}
                    disabled={cancelSwap.isPending}
                  >
                    <XCircle className="w-4 h-4" /> {t('approvals.cancel_approval')}
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        ))
      )}
    </div>
  );
};

// ─── Main Approval Dashboard ───────────────────────────────────────────────────
export const ApprovalDashboard = () => {
  const { t } = useTranslation();
  const { user, adminSelectedDepartmentId, managerSelectedDepartmentId } = useAuthStore();
  const selectedDeptId = user?.role === 'admin' ? adminSelectedDepartmentId : (user?.role === 'manager' ? managerSelectedDepartmentId : null);
  const queryClient = useQueryClient();
  const [expandedLeave, setExpandedLeave] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [swapError, setSwapError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'pending' | 'history'>('pending');

  // Use the rich endpoint for pending leaves (includes employee name, shift, department, TL approvals)
  const { data: pendingLeaves, isLoading: leavesLoading } = useQuery({
    queryKey: ['leaves', 'pending', 'rich', selectedDeptId],
    queryFn: async () => { const res = await api.get('/leaves/pending/rich'); return res.data?.data || []; },
  });

  const { data: pendingSwaps, isLoading: swapsLoading } = useQuery({
    queryKey: ['swaps', 'pending', selectedDeptId],
    queryFn: async () => { const res = await api.get(user?.role === 'manager' ? '/swaps/pending/manager' : '/swaps/pending'); return res.data?.data || []; },
  });

  const { data: pendingItemRequests, isLoading: itemRequestsLoading } = useQuery({
    queryKey: ['itemRequests', 'pending', selectedDeptId],
    queryFn: async () => { const res = await api.get('/item-requests/pending'); return res.data?.data || []; },
  });

  const updateItemRequestStatus = useMutation({
    mutationFn: async ({ id, status }: { id: string, status: string }) => { await api.post(`/item-requests/${id}/status`, { status }); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['itemRequests', 'pending'] }); },
  });

  const approveLeave = useMutation({
    mutationFn: async (leaveId: string) => {
      const endpoint = user?.role === 'team_leader' ? `/leaves/${leaveId}/approve/team-leader` : `/leaves/${leaveId}/approve/manager`;
      await api.post(endpoint);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['leaves'] }); setExpandedLeave(null); },
  });

  const rejectLeave = useMutation({
    mutationFn: async (leaveId: string) => { await api.post(`/leaves/${leaveId}/reject`, { reason: rejectReason }); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['leaves'] }); setRejectReason(''); setExpandedLeave(null); },
  });

  const approveSwap = useMutation({
    mutationFn: async (swapId: string) => { setSwapError(null); await api.post(`/swaps/${swapId}/approve`); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['swaps'] }); setSwapError(null); },
    onError: (err: any) => setSwapError(err?.response?.data?.error || err?.message || t('approvals.failed_approve_swap')),
  });

  const rejectSwap = useMutation({
    mutationFn: async (swapId: string) => { setSwapError(null); await api.post(`/swaps/${swapId}/reject`, { reason: '' }); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['swaps'] }); setSwapError(null); },
    onError: (err: any) => setSwapError(err?.response?.data?.error || err?.message || t('approvals.failed_reject_swap')),
  });

  // Helper: check if shift code looks like a night shift
  const isNightShift = (shiftCode: string) => {
    const lower = (shiftCode || '').toLowerCase();
    return lower.includes('night') || lower.includes('n') || lower === 'c' || lower.includes('ليل');
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-2 sm:gap-3">
            <ShieldCheck className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            {t('approvals.approval_center')}
          </h2>
          <p className="text-sm sm:text-base text-muted-foreground">{t('approvals.review_manage')}</p>
        </div>

        <div className="flex gap-2">
          <Button variant={activeTab === 'pending' ? 'default' : 'outline'} onClick={() => setActiveTab('pending')} className="gap-2">
            <ShieldCheck className="w-4 h-4" /> {t('approvals.pending')}
            {(pendingLeaves?.length || 0) + (pendingSwaps?.length || 0) + (pendingItemRequests?.length || 0) > 0 && (
              <span className="ml-1 px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-white/20">
                {(pendingLeaves?.length || 0) + (pendingSwaps?.length || 0) + (pendingItemRequests?.length || 0)}
              </span>
            )}
          </Button>
          <Button variant={activeTab === 'history' ? 'default' : 'outline'} onClick={() => setActiveTab('history')} className="gap-2">
            <History className="w-4 h-4" /> {t('approvals.history')}
          </Button>
        </div>
      </div>

      {swapError && (
        <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">{swapError}</div>
      )}

      {activeTab === 'history' && (
        <div className="grid lg:grid-cols-2 gap-6">
          <div className="space-y-4">
            <h3 className="text-lg sm:text-xl font-semibold text-foreground flex items-center gap-2">
              <History className="w-5 h-5 text-muted-foreground" />
              {t('approvals.leave_history')}
            </h3>
            <LeaveHistory />
          </div>
          <div className="space-y-4">
            <h3 className="text-lg sm:text-xl font-semibold text-foreground flex items-center gap-2">
              <History className="w-5 h-5 text-muted-foreground" />
              {t('approvals.swap_history')}
            </h3>
            <SwapHistory />
          </div>
        </div>
      )}

      {activeTab === 'pending' && (
        <div className="grid xl:grid-cols-3 lg:grid-cols-2 gap-6">
          {/* ─── Leave Requests ─── */}
          <div className="space-y-4">
            <h3 className="text-lg sm:text-xl font-semibold text-foreground flex items-center gap-2">
              <CalendarOff className="w-5 h-5 text-destructive" />
              {t('approvals.leave_requests')}
              {pendingLeaves?.length > 0 && (
                <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-bold bg-destructive/20 text-destructive border border-destructive/30">
                  {pendingLeaves.length}
                </span>
              )}
            </h3>

            {leavesLoading ? (
              <div className="space-y-3">{[1,2].map(i => <Card key={i} className="animate-pulse h-32" />)}</div>
            ) : pendingLeaves?.length > 0 ? (
              pendingLeaves.map((leave: any) => (
                <Card key={leave.id} className="overflow-hidden">
                  <CardHeader className="pb-2">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <CardTitle className="text-base flex items-center gap-2 flex-wrap">
                          {leave.employee_profile_image && (
                            <img src={leave.employee_profile_image} alt="" className="w-6 h-6 rounded-full object-cover shadow-sm" />
                          )}
                          {leave.employee_name}
                          <span className="text-xs text-muted-foreground font-mono">({leave.employee_code})</span>
                        </CardTitle>
                        <CardDescription className="mt-1">
                          {leave.leave_type_name_en?.charAt(0).toUpperCase() + leave.leave_type_name_en?.slice(1) || 'Leave'} ·{' '}
                          {leave.leave_type_name_en?.toLowerCase() === 'hourly' ? (
                            <>
                              {fmtDate(leave.start_date, 'MMM d, yyyy')}
                              {leave.start_time && leave.end_time && (
                                <> · {leave.start_time} → {leave.end_time}</>
                              )}
                            </>
                          ) : (
                            <>
                              {fmtDate(leave.start_date, 'MMM d')} → {fmtDate(leave.end_date, 'MMM d, yyyy')} ·{' '}
                              {leave.total_days} day{leave.total_days > 1 ? 's' : ''}
                            </>
                          )}
                        </CardDescription>
                      </div>
                      {/* Multi-TL progress */}
                      {leave.total_tls > 1 && (
                        <div className="text-center px-2 py-1 rounded-lg bg-blue-500/10 border border-blue-500/20 flex-shrink-0">
                          <div className="text-sm font-bold text-blue-500">{leave.tl_approvals}/{leave.total_tls}</div>
                          <div className="text-[9px] text-blue-400 uppercase">{t('approvals.tl_approved_count')}</div>
                        </div>
                      )}
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {/* Employee info badges */}
                    <div className="flex flex-wrap gap-2 text-xs">
                      {leave.shift_name && (
                        <span className="px-2 py-1 rounded-md bg-primary/10 text-primary border border-primary/20">
                          🔄 {leave.shift_name} ({leave.shift_code})
                        </span>
                      )}
                      {leave.department_name && (
                        <span className="px-2 py-1 rounded-md bg-muted text-muted-foreground border border-border">
                          🏢 {leave.department_name}
                        </span>
                      )}
                    </div>

                    {leave.reason && <p className="text-sm text-muted-foreground italic">"{leave.reason}"</p>}

                    {/* Shift Coverage for the leave dates */}
                    {leave.default_shift_id && (
                      <CoveragePreview shiftId={leave.default_shift_id} date={leave.start_date?.split('T')[0]} />
                    )}

                    {/* Night shift: show previous day coverage */}
                    {leave.default_shift_id && isNightShift(leave.shift_code) && (
                      <NightShiftPrevDayInfo shiftId={leave.default_shift_id} date={leave.start_date?.split('T')[0]} />
                    )}

                    {expandedLeave === leave.id && (
                      <div className="space-y-3 pt-2 border-t border-border">
                        <QuickReplacementSelect date={leave.start_date?.split('T')[0]} onSelect={(empId) => console.log('Replacement:', empId)} />
                        <div className="space-y-2">
                          <Label className="text-destructive">{t('approvals.rejection_reason_if_rejecting')}</Label>
                          <Input value={rejectReason} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRejectReason(e.target.value)} placeholder={t('approvals.enter_rejection_reason')} />
                        </div>
                      </div>
                    )}
                  </CardContent>
                  <CardFooter className="flex justify-end gap-2 sm:gap-3 pt-0 flex-wrap">
                    {expandedLeave !== leave.id ? (
                      <Button variant="outline" size="sm" onClick={() => setExpandedLeave(leave.id)}>{t('approvals.review_details')}</Button>
                    ) : (
                      <>
                        <Button variant="outline" size="sm" className="text-destructive border-destructive/30 hover:bg-destructive/10 gap-1"
                          onClick={() => rejectLeave.mutate(leave.id)} disabled={!rejectReason || rejectLeave.isPending}>
                          <XCircle className="w-4 h-4" /> {t('approvals.reject')}
                        </Button>
                        <Button size="sm" className="gap-1" onClick={() => approveLeave.mutate(leave.id)} disabled={approveLeave.isPending}>
                          <CheckCircle2 className="w-4 h-4" /> {t('approvals.approve')}
                        </Button>
                      </>
                    )}
                  </CardFooter>
                </Card>
              ))
            ) : (
              <Card className="border-dashed">
                <CardContent className="flex flex-col items-center justify-center p-10 text-muted-foreground">
                  <CalendarOff className="w-10 h-10 mb-3 opacity-20" />
                  <p>{t('approvals.no_pending_leaves')}</p>
                </CardContent>
              </Card>
            )}
          </div>

          {/* ── Swap Requests ── */}
          <div className="space-y-4">
            <h3 className="text-lg sm:text-xl font-semibold text-foreground flex items-center gap-2">
              <ArrowLeftRight className="w-5 h-5 text-primary" />
              {t('approvals.swap_requests')}
              {pendingSwaps?.length > 0 && (
                <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-bold bg-primary/20 text-primary border border-primary/30">
                  {pendingSwaps.length}
                </span>
              )}
            </h3>

            {swapsLoading ? (
              <div className="space-y-3">{[1,2].map(i => <Card key={i} className="animate-pulse h-24" />)}</div>
            ) : pendingSwaps?.length > 0 ? (
              pendingSwaps.map((swap: any) => (
                <Card key={swap.id}>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-base">Shift Swap · {fmtDate(swap.shift_date)}</CardTitle>
                    <CardDescription className="flex items-center gap-2 flex-wrap mt-1">
                      <span className="flex items-center gap-1">
                        Requester
                        {swap.requester_profile_image && <img src={swap.requester_profile_image} alt="" className="w-4 h-4 rounded-full object-cover" />}
                        {swap.requester_name || `#${swap.requester_id?.slice(0, 8)}`}
                      </span>
                      ↔
                      <span className="flex items-center gap-1">
                        Target
                        {swap.target_profile_image && <img src={swap.target_profile_image} alt="" className="w-4 h-4 rounded-full object-cover" />}
                        {swap.target_employee_name || `#${swap.target_employee_id?.slice(0, 8)}`}
                      </span>
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    {swap.reason && <p className="text-sm text-muted-foreground italic">"{swap.reason}"</p>}
                    {swap.shift_id && <div className="mt-3"><CoveragePreview shiftId={swap.shift_id} date={swap.shift_date?.split('T')[0]} /></div>}
                  </CardContent>
                  <CardFooter className="flex justify-end gap-2 sm:gap-3 pt-0">
                    <Button variant="outline" size="sm" className="text-destructive border-destructive/30 hover:bg-destructive/10 gap-1"
                      onClick={() => rejectSwap.mutate(swap.id)} disabled={rejectSwap.isPending}>
                      <XCircle className="w-4 h-4" /> {t('approvals.reject')}
                    </Button>
                    <Button size="sm" className="gap-1" onClick={() => approveSwap.mutate(swap.id)} disabled={approveSwap.isPending}>
                      <CheckCircle2 className="w-4 h-4" /> {t('approvals.approve')}
                    </Button>
                  </CardFooter>
                </Card>
              ))
            ) : (
              <Card className="border-dashed">
                <CardContent className="flex flex-col items-center justify-center p-10 text-muted-foreground">
                  <ArrowLeftRight className="w-10 h-10 mb-3 opacity-20" />
                  <p>{t('approvals.no_pending_swaps')}</p>
                </CardContent>
              </Card>
            )}
          </div>

          {/* ─── Item Requests ─── */}
          <div className="space-y-4">
            <h3 className="text-lg sm:text-xl font-semibold text-foreground flex items-center gap-2">
              <Package className="w-5 h-5 text-indigo-500" />
              {t('approvals.item_requests')}
              {pendingItemRequests?.length > 0 && (
                <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-bold bg-indigo-500/20 text-indigo-600 border border-indigo-500/30">
                  {pendingItemRequests.length}
                </span>
              )}
            </h3>

            {itemRequestsLoading ? (
              <div className="space-y-3">{[1,2].map(i => <Card key={i} className="animate-pulse h-24" />)}</div>
            ) : pendingItemRequests?.length > 0 ? (
              pendingItemRequests.map((req: any) => (
                <Card key={req.id}>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-base">{req.category_name || t('approvals.request')}</CardTitle>
                    <CardDescription className="flex items-center gap-2 flex-wrap mt-1">
                      <span className="flex items-center gap-1">
                        {req.employee_name}
                      </span>
                      • {fmtDate(req.created_at)}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm">{req.description}</p>
                  </CardContent>
                  <CardFooter className="flex justify-end gap-2 sm:gap-3 pt-0">
                    <Button variant="outline" size="sm" className="text-destructive border-destructive/30 hover:bg-destructive/10 gap-1"
                      onClick={() => updateItemRequestStatus.mutate({ id: req.id, status: 'rejected' })} disabled={updateItemRequestStatus.isPending}>
                      <XCircle className="w-4 h-4" /> {t('approvals.reject')}
                    </Button>
                    <Button size="sm" className="bg-indigo-600 hover:bg-indigo-700 gap-1" onClick={() => updateItemRequestStatus.mutate({ id: req.id, status: 'processed' })} disabled={updateItemRequestStatus.isPending}>
                      <CheckCircle2 className="w-4 h-4" /> {t('approvals.processed')}
                    </Button>
                  </CardFooter>
                </Card>
              ))
            ) : (
              <Card className="border-dashed">
                <CardContent className="flex flex-col items-center justify-center p-10 text-muted-foreground">
                  <Package className="w-10 h-10 mb-3 opacity-20" />
                  <p>{t('approvals.no_pending_items')}</p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
