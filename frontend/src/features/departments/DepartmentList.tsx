import { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Building2, Crown, Plus, Edit3, Trash2, Save, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Link } from 'react-router-dom';

interface Department {
  id: string;
  department_code: string;
  name: string;
  description: string | null;
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
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const queryClient = useQueryClient();

  // ── create form state ──────────────────────────────────────────────
  const [deptCode, setDeptCode] = useState('');
  const [deptName, setDeptName] = useState('');
  const [deptDesc, setDeptDesc] = useState('');
  // single dropdown; we wrap it in an array before sending
  const [deptManagerId, setDeptManagerId] = useState<string>('');
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  // ── edit form state ────────────────────────────────────────────────
  const [editId, setEditId] = useState<string | null>(null);
  const [_editCode, setEditCode] = useState('');
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
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
        // backend expects manager_ids array
        manager_ids: deptManagerId ? [deptManagerId] : [],
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['departments'] });
      setDeptCode(''); setDeptName(''); setDeptDesc(''); setDeptManagerId('');
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to create department');
    },
  });

  const updateDepartment = useMutation({
    mutationFn: async () => {
      if (!editId) return;
      setError(null);
      await api.put(`/departments/${editId}`, {
        name: editName,
        description: editDesc || null,
        // send the selected manager wrapped in array (replaces current list)
        manager_ids: editManagerId ? [editManagerId] : [],
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['departments'] });
      setEditId(null);
    },
    onError: (err: any) =>
      setError(err?.response?.data?.error || err?.message || 'Failed to update department'),
  });

  const deleteDepartment = useMutation({
    mutationFn: async (id: string) => {
      setError(null);
      await api.delete(`/departments/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['departments'] }),
    onError: (err: any) =>
      setError(err?.response?.data?.error || err?.message || 'Failed to delete department'),
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
    if (!firstId || !managerMap[firstId]) return 'Unassigned';
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
          Departments
        </h2>
        <p className="text-muted-foreground">View and manage all company departments.</p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="grid md:grid-cols-3 gap-4">
            <div className="space-y-2 md:col-span-2">
              <Label>Search</Label>
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by name or department code..."
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
              Create Department
            </CardTitle>
            <CardDescription>Create a department and assign its manager</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {error && (
              <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-sm">
                {error}
              </div>
            )}
            <div className="grid md:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>Department Code</Label>
                <Input value={deptCode} onChange={(e) => setDeptCode(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Department Name</Label>
                <Input value={deptName} onChange={(e) => setDeptName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Manager</Label>
                <select
                  className={selectClass}
                  value={deptManagerId}
                  onChange={(e) => setDeptManagerId(e.target.value)}
                >
                  <option value="">Unassigned</option>
                  {managers?.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.first_name} {m.last_name} — {m.employee_code}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-2 md:col-span-3">
                <Label>Description</Label>
                <Input value={deptDesc} onChange={(e) => setDeptDesc(e.target.value)} />
              </div>
            </div>
          </CardContent>
          <CardFooter>
            <Button
              onClick={() => createDepartment.mutate()}
              disabled={createDepartment.isPending || !deptCode || !deptName}
              className="gap-2"
            >
              <Plus className="w-4 h-4" /> Create
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
          Failed to load departments. Please try again.
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
                        <Label className="text-xs">Name</Label>
                        <Input
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          className="h-9"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-xs">Manager</Label>
                        <select
                          className={selectClass}
                          value={editManagerId}
                          onChange={(e) => setEditManagerId(e.target.value)}
                        >
                          <option value="">Unassigned</option>
                          {managers?.map((m) => (
                            <option key={m.id} value={m.id}>
                              {m.first_name} {m.last_name}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="space-y-2">
                        <Label className="text-xs">Description</Label>
                        <Input
                          value={editDesc}
                          onChange={(e) => setEditDesc(e.target.value)}
                          className="h-9"
                        />
                      </div>
                      <div className="flex justify-end gap-2 pt-1">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={(e) => { e.preventDefault(); setEditId(null); }}
                        >
                          <X className="w-4 h-4 mr-1" /> Cancel
                        </Button>
                        <Button
                          size="sm"
                          onClick={(e) => { e.preventDefault(); updateDepartment.mutate(); }}
                          disabled={updateDepartment.isPending || !editName}
                        >
                          <Save className="w-4 h-4 mr-1" /> Save
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
                        <span>Manager:</span>
                        <span className="text-foreground font-medium">
                          {primaryManagerLabel(dept)}
                        </span>
                      </div>
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
                        // pre-fill with first assigned manager (if any)
                        setEditManagerId(dept.manager_ids?.[0] || '');
                      }}
                    >
                      <Edit3 className="w-4 h-4 mr-1" /> Edit
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-destructive border-destructive/30 hover:bg-destructive/10"
                      onClick={() => {
                        if (confirm(`Delete department "${dept.name}"?`))
                          deleteDepartment.mutate(dept.id);
                      }}
                    >
                      <Trash2 className="w-4 h-4 mr-1" /> Delete
                    </Button>
                  </CardFooter>
                )}
              </Card>
            </Link>
          ))}
          {filteredDepartments?.length === 0 && (
            <div className="col-span-full p-8 text-center text-muted-foreground border border-border border-dashed rounded-xl bg-muted/10">
              No departments match current filters.
            </div>
          )}
        </div>
      )}
    </div>
  );
};
