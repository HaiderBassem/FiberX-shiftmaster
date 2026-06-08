import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Shield, X, User, Save, Building2 } from 'lucide-react';
import api from '@/lib/api';
import { fiberxDataService } from '../../services/fiberxDataService';
import { Button } from '@/components/ui/button';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  documentId: string;
}

export function FiberxDataPermissionsModal({ isOpen, onClose, documentId }: Props) {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<'employees' | 'departments'>('employees');
  
  // Employee state
  const [selectedEmployee, setSelectedEmployee] = useState('');
  const [accessLevel, setAccessLevel] = useState('read');
  
  // Department state
  const [selectedDepartment, setSelectedDepartment] = useState('');
  const [deptAccessLevel, setDeptAccessLevel] = useState('read');

  // Queries
  const { data: employees = [] } = useQuery({
    queryKey: ['employees'],
    queryFn: async () => {
      const res = await api.get('/employees');
      return res.data?.data || [];
    },
    enabled: isOpen,
  });

  const { data: departments = [] } = useQuery({
    queryKey: ['departments'],
    queryFn: async () => {
      const res = await api.get('/departments');
      return res.data?.data || [];
    },
    enabled: isOpen,
  });

  const { data: employeeAccess = [], isLoading: loadingAccess } = useQuery({
    queryKey: ['fiberx-data-access', documentId],
    queryFn: () => fiberxDataService.getEmployeeAccessList(documentId),
    enabled: isOpen && !!documentId,
  });

  const { data: departmentShares = [], isLoading: loadingShares } = useQuery({
    queryKey: ['fiberx-data-shares', documentId],
    queryFn: () => fiberxDataService.getDepartmentShares(documentId),
    enabled: isOpen && !!documentId,
  });

  // Mutations
  const updateAccessMutation = useMutation({
    mutationFn: (data: { employee_id: string; access_level: string }) =>
      fiberxDataService.setEmployeeAccess(documentId, data.employee_id, data.access_level),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fiberx-data-access', documentId] });
      queryClient.invalidateQueries({ queryKey: ['fiberx-data'] });
      setSelectedEmployee('');
    },
  });

  const updateShareMutation = useMutation({
    mutationFn: (data: { department_id: string; access_level: string }) =>
      fiberxDataService.setDepartmentShare(documentId, data.department_id, data.access_level),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fiberx-data-shares', documentId] });
      queryClient.invalidateQueries({ queryKey: ['fiberx-data'] });
      setSelectedDepartment('');
    },
  });

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-0">
      <div className="absolute inset-0 bg-background/80 backdrop-blur-sm" onClick={onClose} />
      
      <div className="relative w-full max-w-2xl bg-card rounded-2xl shadow-xl border border-border flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary/10 text-primary rounded-xl">
              <Shield className="w-6 h-6" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-foreground">Data Permissions</h2>
              <p className="text-sm text-muted-foreground">Manage who can access this document</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-muted-foreground hover:bg-muted rounded-full transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-border">
          <button
            onClick={() => setActiveTab('employees')}
            className={`flex-1 py-3 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'employees' 
                ? 'border-primary text-primary' 
                : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'
            }`}
          >
            Employee Access
          </button>
          <button
            onClick={() => setActiveTab('departments')}
            className={`flex-1 py-3 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'departments' 
                ? 'border-primary text-primary' 
                : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'
            }`}
          >
            Department Sharing
          </button>
        </div>

        <div className="p-6 overflow-y-auto">
          {activeTab === 'employees' ? (
            <div className="space-y-6">
              {/* Add New Employee Access */}
              <div className="bg-muted/50 rounded-xl p-4 border border-border">
                <h3 className="text-sm font-medium text-foreground mb-3">Add Exception</h3>
                <div className="flex gap-3">
                  <select
                    value={selectedEmployee}
                    onChange={(e) => setSelectedEmployee(e.target.value)}
                    className="flex-1 px-3 py-2 bg-background border border-input rounded-lg focus:ring-2 focus:ring-primary focus:border-primary text-sm"
                  >
                    <option value="">Select employee...</option>
                    {employees.map((emp: any) => (
                      <option key={emp.id} value={emp.id}>
                        {emp.first_name} {emp.last_name} ({emp.department_id ? 'Has Dept' : 'No Dept'})
                      </option>
                    ))}
                  </select>
                  
                  <select
                    value={accessLevel}
                    onChange={(e) => setAccessLevel(e.target.value)}
                    className="w-32 px-3 py-2 bg-background border border-input rounded-lg focus:ring-2 focus:ring-primary focus:border-primary text-sm"
                  >
                    <option value="read">Read Only</option>
                    <option value="write">Editor</option>
                    <option value="hide">Hide (Deny)</option>
                  </select>

                  <Button
                    onClick={() => {
                      if (selectedEmployee) {
                        updateAccessMutation.mutate({ employee_id: selectedEmployee, access_level: accessLevel });
                      }
                    }}
                    disabled={!selectedEmployee || updateAccessMutation.isPending}
                    className="gap-2"
                  >
                    <Save className="w-4 h-4" />
                    Save
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  "Hide" completely removes access to this document for the specific employee, even if their department has access.
                </p>
              </div>

              {/* Current Exceptions List */}
              <div>
                <h3 className="text-sm font-medium text-foreground mb-3">Current Exceptions</h3>
                {loadingAccess ? (
                  <p className="text-sm text-muted-foreground">Loading...</p>
                ) : employeeAccess.length === 0 ? (
                  <div className="text-center py-6 bg-muted/30 rounded-lg border border-dashed border-border">
                    <User className="w-8 h-8 text-muted-foreground mx-auto mb-2 opacity-50" />
                    <p className="text-sm text-muted-foreground">No employee exceptions configured.</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {employeeAccess.map((access) => (
                      <div key={access.id} className="flex items-center justify-between p-3 bg-background border border-border rounded-lg">
                        <div className="flex items-center gap-3">
                          <div className={`p-2 rounded-lg ${
                            access.access_level === 'hide' ? 'bg-destructive/10 text-destructive' :
                            access.access_level === 'write' ? 'bg-amber-500/10 text-amber-600' :
                            'bg-primary/10 text-primary'
                          }`}>
                            <User className="w-4 h-4" />
                          </div>
                          <div>
                            <p className="text-sm font-medium text-foreground">{access.employee_name}</p>
                            <p className="text-xs text-muted-foreground capitalize">{access.access_level}</p>
                          </div>
                        </div>
                        
                        <div className="flex items-center gap-2">
                          <select
                            value={access.access_level}
                            onChange={(e) => updateAccessMutation.mutate({ 
                              employee_id: access.employee_id, 
                              access_level: e.target.value 
                            })}
                            className="px-2 py-1 bg-background border border-input rounded text-xs"
                            disabled={updateAccessMutation.isPending}
                          >
                            <option value="read">Read</option>
                            <option value="write">Write</option>
                            <option value="hide">Hide</option>
                          </select>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              {/* Share with Department */}
              <div className="bg-muted/50 rounded-xl p-4 border border-border">
                <h3 className="text-sm font-medium text-foreground mb-3">Share with Department</h3>
                <div className="flex gap-3">
                  <select
                    value={selectedDepartment}
                    onChange={(e) => setSelectedDepartment(e.target.value)}
                    className="flex-1 px-3 py-2 bg-background border border-input rounded-lg focus:ring-2 focus:ring-primary focus:border-primary text-sm"
                  >
                    <option value="">Select department...</option>
                    {departments.map((dept: any) => (
                      <option key={dept.id} value={dept.id}>
                        {dept.name}
                      </option>
                    ))}
                  </select>
                  
                  <select
                    value={deptAccessLevel}
                    onChange={(e) => setDeptAccessLevel(e.target.value)}
                    className="w-32 px-3 py-2 bg-background border border-input rounded-lg focus:ring-2 focus:ring-primary focus:border-primary text-sm"
                  >
                    <option value="read">Read Only</option>
                    <option value="write">Editor</option>
                    <option value="none">Remove Share</option>
                  </select>

                  <Button
                    onClick={() => {
                      if (selectedDepartment) {
                        updateShareMutation.mutate({ department_id: selectedDepartment, access_level: deptAccessLevel });
                      }
                    }}
                    disabled={!selectedDepartment || updateShareMutation.isPending}
                    className="gap-2"
                  >
                    <Save className="w-4 h-4" />
                    Save
                  </Button>
                </div>
              </div>

              {/* Current Shares List */}
              <div>
                <h3 className="text-sm font-medium text-foreground mb-3">Current Shares</h3>
                {loadingShares ? (
                  <p className="text-sm text-muted-foreground">Loading...</p>
                ) : departmentShares.length === 0 ? (
                  <div className="text-center py-6 bg-muted/30 rounded-lg border border-dashed border-border">
                    <Building2 className="w-8 h-8 text-muted-foreground mx-auto mb-2 opacity-50" />
                    <p className="text-sm text-muted-foreground">Not shared with any other departments.</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {departmentShares.map((share) => (
                      <div key={share.id} className="flex items-center justify-between p-3 bg-background border border-border rounded-lg">
                        <div className="flex items-center gap-3">
                          <div className={`p-2 rounded-lg ${
                            share.access_level === 'write' ? 'bg-amber-500/10 text-amber-600' : 'bg-primary/10 text-primary'
                          }`}>
                            <Building2 className="w-4 h-4" />
                          </div>
                          <div>
                            <p className="text-sm font-medium text-foreground">{share.department_name}</p>
                            <p className="text-xs text-muted-foreground capitalize">{share.access_level}</p>
                          </div>
                        </div>
                        
                        <div className="flex items-center gap-2">
                          <select
                            value={share.access_level}
                            onChange={(e) => updateShareMutation.mutate({ 
                              department_id: share.department_id, 
                              access_level: e.target.value 
                            })}
                            className="px-2 py-1 bg-background border border-input rounded text-xs"
                            disabled={updateShareMutation.isPending}
                          >
                            <option value="read">Read Only</option>
                            <option value="write">Editor</option>
                            <option value="none">Remove</option>
                          </select>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
