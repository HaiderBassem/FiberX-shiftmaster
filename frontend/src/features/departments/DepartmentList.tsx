import { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Building2, Crown, Plus, Edit3, Trash2, Save, X, Database } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

interface Department {
  id: string;
  department_code: string;
  name: string;
  description: string | null;
  fiberx_enabled: boolean;
  max_leaves_per_day?: number | null;
  max_hourly_leaves_per_day?: number | null;
  manager_ids: string[]; // now an array (multi-manager support)
  created_at: string;
}

type Employee = {
  id: string;
  employee_code: string;
  first_name: string;
  last_name: string;
  email: string;
  role: string;
};

export const DepartmentList = () => {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const queryClient = useQueryClient();

  // ── create form state ──────────────────────────────────────────────
  const [deptCode, setDeptCode] = useState('');
  const [deptName, setDeptName] = useState('');
  const [deptDesc, setDeptDesc] = useState('');
  // single dropdown; we wrap it in an array before sending
  const [deptManagerId, setDeptManagerId] = useState<string>('');
  const [deptMaxLeaves, setDeptMaxLeaves] = useState<string>('');
  const [deptMaxHourlyLeaves, setDeptMaxHourlyLeaves] = useState<string>('');
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  // ── edit form state ────────────────────────────────────────────────
  const [editId, setEditId] = useState<string | null>(null);
  const [_editCode, setEditCode] = useState('');
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editMaxLeaves, setEditMaxLeaves] = useState('');
  const [editMaxHourlyLeaves, setEditMaxHourlyLeaves] = useState('');
  const [editManagerId, setEditManagerId] = useState('');

  // ── queries ────────────────────────────────────────────────────────
  const { data: departments, isLoading, isError } = useQuery<Department[]>({
    queryKey: ['departments'],
    queryFn: async () => {
      const response = await api.get('/departments');
      return response.data?.data || [];
    },
  });

  const { data: managers } = useQuery<Employee[]>({
    queryKey: ['employees', 'role', 'manager'],
    queryFn: async () => {
      const res = await api.get('/employees?role=manager');
      return res.data?.data || [];
    },
    enabled: isAdmin,
  });

  const managerMap = useMemo(() => {
    const m: Record<string, Employee> = {};
    (managers || []).forEach((e) => { m[e.id] = e; });
    return m;
  }, [managers]);

  // ── mutations ──────────────────────────────────────────────────────
  const createDepartment = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.post('/departments', {
        department_code: deptCode,
        name: deptName,
        description: deptDesc || null,
        max_leaves_per_day: deptMaxLeaves ? parseInt(deptMaxLeaves, 10) : null,
        max_hourly_leaves_per_day: deptMaxHourlyLeaves ? parseInt(deptMaxHourlyLeaves, 10) : null,
        // backend expects manager_ids array
        manager_ids: deptManagerId ? [deptManagerId] : [],
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['departments'] });
      setDeptCode(''); setDeptName(''); setDeptDesc(''); setDeptManagerId(''); setDeptMaxLeaves(''); setDeptMaxHourlyLeaves('');
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || t('departments.failed_create'));
    },
  });

  const updateDepartment = useMutation({
    mutationFn: async () => {
      if (!editId) return;
      setError(null);
      await api.put(`/departments/${editId}`, {
        name: editName,
        description: editDesc || null,
        max_leaves_per_day: editMaxLeaves ? parseInt(editMaxLeaves, 10) : null,
        max_hourly_leaves_per_day: editMaxHourlyLeaves ? parseInt(editMaxHourlyLeaves, 10) : null,
        // send the selected manager wrapped in array (replaces current list)
        manager_ids: editManagerId ? [editManagerId] : [],
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['departments'] });
      setEditId(null);
    },
    onError: (err: any) =>
      setError(err?.response?.data?.error || err?.message || t('departments.failed_update')),
  });

  const deleteDepartment = useMutation({
    mutationFn: async (id: string) => {
      setError(null);
      await api.delete(`/departments/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['departments'] }),
    onError: (err: any) =>
      setError(err?.response?.data?.error || err?.message || t('departments.failed_delete')),
  });

  const filteredDepartments = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return departments || [];
    return (departments || []).filter((d) =>
      d.name.toLowerCase().includes(q) || d.department_code.toLowerCase().includes(q),
    );
  }, [departments, search]);

  // ── helpers ────────────────────────────────────────────────────────
  const selectClass =
    'w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors';

  /** Returns the first manager's display name, or "Unassigned" */
  const primaryManagerLabel = (dept: Department) => {
    const firstId = dept.manager_ids?.[0];
    if (!firstId || !managerMap[firstId]) return t('departments.unassigned');
    const m = managerMap[firstId];
    const extra = (dept.manager_ids?.length ?? 0) > 1 ? ` +${dept.manager_ids.length - 1}` : '';
    return `${m.first_name} ${m.last_name}${extra}`;
  };

  // ── render ─────────────────────────────────────────────────────────
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:gap-3">
          <Building2 className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
          {t('departments.title')}
        </h2>
        <p className="text-muted-foreground">{t('departments.description')}</p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="grid md:grid-cols-3 gap-4">
            <div className="space-y-2 md:col-span-2">
              <Label>{t('departments.search_label')}</Label>
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('departments.search_placeholder')}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {isAdmin && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Plus className="w-5 h-5 text-primary" />
              {t('departments.create')}
            </CardTitle>
            <CardDescription>{t('departments.create_desc')}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {error && (
              <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-sm">
                {error}
              </div>
            )}
            <div className="grid md:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>{t('departments.code')}</Label>
                <Input value={deptCode} onChange={(e) => setDeptCode(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>{t('departments.name')}</Label>
                <Input value={deptName} onChange={(e) => setDeptName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>{t('departments.manager')}</Label>
                <select
                  className={selectClass}
                  value={deptManagerId}
                  onChange={(e) => setDeptManagerId(e.target.value)}
                >
                  <option value="">{t('departments.unassigned')}</option>
                  {managers?.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.first_name} {m.last_name} — {m.employee_code}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-2 md:col-span-2">
                <Label>{t('departments.desc')}</Label>
                <Input value={deptDesc} onChange={(e) => setDeptDesc(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>{t('departments.max_leaves')}</Label>
                <Input type="number" min="1" value={deptMaxLeaves} onChange={(e) => setDeptMaxLeaves(e.target.value)} placeholder={t('departments.no_limit')} />
              </div>
              <div className="space-y-2">
                <Label>{t('departments.max_hourly_leaves')}</Label>
                <Input type="number" min="1" value={deptMaxHourlyLeaves} onChange={(e) => setDeptMaxHourlyLeaves(e.target.value)} placeholder={t('departments.no_limit')} />
              </div>
            </div>
          </CardContent>
          <CardFooter>
            <Button
              onClick={() => createDepartment.mutate()}
              disabled={createDepartment.isPending || !deptCode || !deptName}
              className="gap-2"
            >
              <Plus className="w-4 h-4" /> {t('departments.btn_create')}
            </Button>
          </CardFooter>
        </Card>
      )}

      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader className="h-24 bg-muted/50 rounded-t-xl" />
              <CardContent className="h-16" />
            </Card>
          ))}
        </div>
      ) : isError ? (
        <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive">
          {t('departments.failed_load')}
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {filteredDepartments?.map((dept) => (
            <Link key={dept.id} to={`/departments/${dept.id}`} className="block">
              <Card className="hover:shadow-md transition-all">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <div className="space-y-1">
                    <CardTitle className="text-lg font-medium">{dept.name}</CardTitle>
                    <CardDescription className="text-xs font-mono">
                      {dept.department_code}
                    </CardDescription>
                  </div>
                  <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center text-primary">
                    <Building2 className="h-5 w-5" />
                  </div>
                </CardHeader>

                <CardContent className="space-y-2">
                  {editId === dept.id ? (
                    <div className="space-y-3 pt-1" onClick={(e) => e.preventDefault()}>
                      <div className="space-y-2">
                        <Label className="text-xs">{t('departments.name')}</Label>
                        <Input
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          className="h-9"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-xs">{t('departments.manager')}</Label>
                        <select
                          className={selectClass}
                          value={editManagerId}
                          onChange={(e) => setEditManagerId(e.target.value)}
                        >
                          <option value="">{t('departments.unassigned')}</option>
                          {managers?.map((m) => (
                            <option key={m.id} value={m.id}>
                              {m.first_name} {m.last_name}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label className="text-xs">{t('departments.desc')}</Label>
                          <Input
                            value={editDesc}
                            onChange={(e) => setEditDesc(e.target.value)}
                            className="h-9"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-xs">{t('departments.max_leaves')}</Label>
                          <Input
                            type="number"
                            min="1"
                            value={editMaxLeaves}
                            onChange={(e) => setEditMaxLeaves(e.target.value)}
                            className="h-9"
                            placeholder={t('departments.no_limit')}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label className="text-xs">{t('departments.max_hourly_leaves')}</Label>
                          <Input
                            type="number"
                            min="1"
                            value={editMaxHourlyLeaves}
                            onChange={(e) => setEditMaxHourlyLeaves(e.target.value)}
                            className="h-9"
                            placeholder={t('departments.no_limit')}
                          />
                        </div>
                      </div>
                      <div className="flex justify-end gap-2 pt-1">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={(e) => { e.preventDefault(); setEditId(null); }}
                        >
                          <X className="w-4 h-4 mr-1" /> {t('departments.btn_cancel')}
                        </Button>
                        <Button
                          size="sm"
                          onClick={(e) => { e.preventDefault(); updateDepartment.mutate(); }}
                          disabled={updateDepartment.isPending || !editName}
                        >
                          <Save className="w-4 h-4 mr-1" /> {t('departments.btn_save')}
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <>
                      {dept.description && (
                        <p className="text-sm text-muted-foreground">{dept.description}</p>
                      )}
                      <div className="text-sm text-muted-foreground flex items-center gap-2">
                        <Crown className="w-4 h-4 text-amber-500/80" />
                        <span>{t('departments.manager')}:</span>
                        <span className="text-foreground font-medium">
                          {primaryManagerLabel(dept)}
                        </span>
                      </div>
                      {isAdmin && (
                        <div 
                          className="flex items-center justify-between p-2.5 mt-2 rounded-lg bg-muted/40 border border-border/50"
                          onClick={(e) => e.preventDefault()}
                        >
                          <div className="flex items-center gap-2">
                            <Database className="w-4 h-4 text-indigo-500" />
                            <span className="text-xs font-medium text-foreground">{t('departments.fiberx_data')}</span>
                          </div>
                          <label className="relative inline-flex items-center cursor-pointer">
                            <input
                              type="checkbox"
                              className="sr-only peer"
                              checked={dept.fiberx_enabled}
                              onChange={(e) => {
                                const newVal = e.target.checked;
                                api.put(`/departments/${dept.id}/fiberx-toggle`, { enabled: newVal }).then(() => {
                                  queryClient.invalidateQueries({ queryKey: ['departments'] });
                                }).catch((err: any) => {
                                  alert(err?.response?.data?.error || t('departments.failed_toggle_fiberx'));
                                  queryClient.invalidateQueries({ queryKey: ['departments'] });
                                });
                              }}
                            />
                            <div className="w-9 h-5 bg-muted peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/20 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
                          </label>
                        </div>
                      )}
                    </>
                  )}
                </CardContent>

                {isAdmin && editId !== dept.id && (
                  <CardFooter
                    className="pt-0 flex justify-end gap-2"
                    onClick={(e) => e.preventDefault()}
                  >
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setEditId(dept.id);
                        setEditCode(dept.department_code);
                        setEditName(dept.name);
                        setEditDesc(dept.description || '');
                        setEditMaxLeaves(dept.max_leaves_per_day ? dept.max_leaves_per_day.toString() : '');
                        setEditMaxHourlyLeaves(dept.max_hourly_leaves_per_day ? dept.max_hourly_leaves_per_day.toString() : '');
                        // pre-fill with first assigned manager (if any)
                        setEditManagerId(dept.manager_ids?.[0] || '');
                      }}
                    >
                      <Edit3 className="w-4 h-4 mr-1" /> {t('departments.btn_edit')}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-destructive border-destructive/30 hover:bg-destructive/10"
                      onClick={() => {
                        if (confirm(t('departments.confirm_delete', { name: dept.name })))
                          deleteDepartment.mutate(dept.id);
                      }}
                    >
                      <Trash2 className="w-4 h-4 mr-1" /> {t('departments.btn_delete')}
                    </Button>
                  </CardFooter>
                )}
              </Card>
            </Link>
          ))}
          {filteredDepartments?.length === 0 && (
            <div className="col-span-full p-8 text-center text-muted-foreground border border-border border-dashed rounded-xl bg-muted/10">
              {t('departments.no_match')}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
