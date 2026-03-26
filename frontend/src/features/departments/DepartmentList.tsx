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
  manager_id: string | null;
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

  const [deptCode, setDeptCode] = useState('');
  const [deptName, setDeptName] = useState('');
  const [deptDesc, setDeptDesc] = useState('');
  const [deptManagerId, setDeptManagerId] = useState<string>('');
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  const [editId, setEditId] = useState<string | null>(null);
  const [editCode, setEditCode] = useState('');
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editManagerId, setEditManagerId] = useState('');

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

  const createDepartment = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.post('/departments', {
        department_code: deptCode,
        name: deptName,
        description: deptDesc || null,
        manager_id: deptManagerId || null,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['departments'] });
      setDeptCode('');
      setDeptName('');
      setDeptDesc('');
      setDeptManagerId('');
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
        department_code: editCode,
        name: editName,
        description: editDesc || null,
        manager_id: editManagerId || null,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['departments'] });
      setEditId(null);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to update department'),
  });

  const deleteDepartment = useMutation({
    mutationFn: async (id: string) => {
      setError(null);
      await api.delete(`/departments/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['departments'] }),
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to delete department'),
  });

  const filteredDepartments = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return departments || [];
    return (departments || []).filter((d) =>
      d.name.toLowerCase().includes(q) || d.department_code.toLowerCase().includes(q),
    );
  }, [departments, search]);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold tracking-tight text-white">Departments</h2>
        <p className="text-zinc-400">View and manage all company departments.</p>
      </div>

      <Card className="bg-zinc-900/40 border-zinc-800/60">
        <CardContent className="pt-6">
          <div className="grid md:grid-cols-3 gap-4">
            <div className="space-y-2 md:col-span-2">
              <Label className="text-zinc-300">Search</Label>
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by name or department code..."
                className="bg-black/20 border-zinc-700"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {isAdmin && (
        <Card className="bg-zinc-900/50 border-zinc-800/60">
          <CardHeader>
            <CardTitle className="text-white flex items-center gap-2">
              <Plus className="w-5 h-5 text-emerald-400" />
              Create Department
            </CardTitle>
            <CardDescription>Create a department and assign its manager</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {error && (
              <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-300 text-sm">
                {error}
              </div>
            )}
            <div className="grid md:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label className="text-zinc-300">Department Code</Label>
                <Input value={deptCode} onChange={(e) => setDeptCode(e.target.value)} className="bg-black/20 border-zinc-700" />
              </div>
              <div className="space-y-2">
                <Label className="text-zinc-300">Department Name</Label>
                <Input value={deptName} onChange={(e) => setDeptName(e.target.value)} className="bg-black/20 border-zinc-700" />
              </div>
              <div className="space-y-2">
                <Label className="text-zinc-300">Manager</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
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
                <Label className="text-zinc-300">Description</Label>
                <Input value={deptDesc} onChange={(e) => setDeptDesc(e.target.value)} className="bg-black/20 border-zinc-700" />
              </div>
            </div>
          </CardContent>
          <CardFooter>
            <Button
              className="bg-emerald-600 hover:bg-emerald-500 text-white gap-2"
              onClick={() => createDepartment.mutate()}
              disabled={createDepartment.isPending || !deptCode || !deptName}
            >
              <Plus className="w-4 h-4" />
              Create
            </Button>
          </CardFooter>
        </Card>
      )}

      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="animate-pulse bg-zinc-900/50">
              <CardHeader className="h-24 bg-zinc-800/50 rounded-t-xl" />
              <CardContent className="h-16" />
            </Card>
          ))}
        </div>
      ) : isError ? (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400">
          Failed to load departments. Please try again.
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {filteredDepartments?.map((dept) => (
            <Link key={dept.id} to={`/departments/${dept.id}`} className="block">
              <Card className="bg-zinc-900/40 hover:bg-zinc-800/60 transition-colors border-zinc-800/60">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <div className="space-y-1">
                    <CardTitle className="text-lg font-medium text-white">{dept.name}</CardTitle>
                    <CardDescription className="text-xs font-mono text-zinc-500">{dept.department_code}</CardDescription>
                  </div>
                  <div className="h-10 w-10 rounded-full bg-blue-500/10 flex items-center justify-center text-blue-400">
                    <Building2 className="h-5 w-5" />
                  </div>
                </CardHeader>
                <CardContent className="space-y-2">
                  {editId === dept.id ? (
                    <div
                      className="space-y-3 pt-1"
                      onClick={(e) => e.preventDefault()}
                    >
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Code</Label>
                        <Input value={editCode} onChange={(e) => setEditCode(e.target.value)} className="bg-black/20 border-zinc-700" />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Name</Label>
                        <Input value={editName} onChange={(e) => setEditName(e.target.value)} className="bg-black/20 border-zinc-700" />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Manager</Label>
                        <select
                          className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
                          value={editManagerId}
                          onChange={(e) => setEditManagerId(e.target.value)}
                        >
                          <option value="">Unassigned</option>
                          {managers?.map((m) => (
                            <option key={m.id} value={m.id}>
                              {m.first_name} {m.last_name} — {m.employee_code}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Description</Label>
                        <Input value={editDesc} onChange={(e) => setEditDesc(e.target.value)} className="bg-black/20 border-zinc-700" />
                      </div>
                      <div className="flex justify-end gap-2 pt-1">
                        <Button
                          size="sm"
                          variant="outline"
                          className="border-zinc-700 text-zinc-300"
                          onClick={(e) => {
                            e.preventDefault();
                            setEditId(null);
                          }}
                        >
                          <X className="w-4 h-4 mr-1" /> Cancel
                        </Button>
                        <Button
                          size="sm"
                          className="bg-emerald-600 hover:bg-emerald-500 text-white"
                          onClick={(e) => {
                            e.preventDefault();
                            updateDepartment.mutate();
                          }}
                          disabled={updateDepartment.isPending || !editCode || !editName}
                        >
                          <Save className="w-4 h-4 mr-1" /> Save
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <>
                      {dept.description && (
                        <p className="text-sm text-zinc-400">{dept.description}</p>
                      )}
                      <div className="text-sm text-zinc-400 flex items-center gap-2">
                        <Crown className="w-4 h-4 text-amber-400/80" />
                        <span className="text-zinc-500">Manager:</span>
                        <span className="text-zinc-200 font-medium">
                          {dept.manager_id && managerMap[dept.manager_id]
                            ? `${managerMap[dept.manager_id].first_name} ${managerMap[dept.manager_id].last_name}`
                            : 'Unassigned'}
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
                      className="border-zinc-700 text-zinc-300 hover:bg-zinc-800"
                      onClick={() => {
                        setEditId(dept.id);
                        setEditCode(dept.department_code);
                        setEditName(dept.name);
                        setEditDesc(dept.description || '');
                        setEditManagerId(dept.manager_id || '');
                      }}
                    >
                      <Edit3 className="w-4 h-4 mr-1" />
                      Edit
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="border-red-500/30 text-red-300 hover:bg-red-500/10"
                      onClick={() => {
                        if (confirm(`Delete department "${dept.name}"?`)) {
                          deleteDepartment.mutate(dept.id);
                        }
                      }}
                    >
                      <Trash2 className="w-4 h-4 mr-1" />
                      Delete
                    </Button>
                  </CardFooter>
                )}
              </Card>
            </Link>
          ))}
          {filteredDepartments?.length === 0 && (
            <div className="col-span-full p-8 text-center text-zinc-500 border border-zinc-800/60 border-dashed rounded-xl bg-zinc-900/20">
              No departments match current filters.
            </div>
          )}
        </div>
      )}
    </div>
  );
};
