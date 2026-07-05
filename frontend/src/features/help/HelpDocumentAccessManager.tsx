import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { helpDocumentService } from '../../services/api/helpDocumentService';
import api from '../../lib/api';
import { Shield, Loader2 } from 'lucide-react';

interface Employee {
  id: string;
  employee_code: string;
  first_name: string;
  last_name: string;
}

export function HelpDocumentAccessManager({ documentId }: { documentId: string }) {
  const queryClient = useQueryClient();
  const [selectedEmployee, setSelectedEmployee] = useState('');
  const [selectedAccess, setSelectedAccess] = useState('read');

  const { data: employees = [], isLoading: loadingEmps } = useQuery<Employee[]>({
    queryKey: ['employees'],
    queryFn: async () => {
      const res = await api.get('/employees');
      return res.data?.data || [];
    },
  });

  const { data: accessList = [], isLoading: loadingAccess } = useQuery({
    queryKey: ['help-docs', documentId, 'access'],
    queryFn: () => helpDocumentService.getAccessList(documentId),
  });

  const setAccessMutation = useMutation({
    mutationFn: (data: { employee_id: string; access_level: string }) => 
      helpDocumentService.setEmployeeAccess(documentId, data.employee_id, data.access_level),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['help-docs', documentId, 'access'] });
      setSelectedEmployee('');
    }
  });

  const handleGrantAccess = () => {
    if (!selectedEmployee) return;
    setAccessMutation.mutate({ employee_id: selectedEmployee, access_level: selectedAccess });
  };

  const handleUpdateAccess = (empId: string, level: string) => {
    setAccessMutation.mutate({ employee_id: empId, access_level: level });
  };

  if (loadingEmps || loadingAccess) return <div className="flex items-center justify-center p-4"><Loader2 className="w-6 h-6 animate-spin text-indigo-600" /></div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 border-b border-gray-100 pb-4">
        <Shield className="w-5 h-5 text-indigo-600" />
        <h3 className="text-lg font-bold text-gray-900">Manage Employee Access</h3>
      </div>

      <div className="bg-gray-50 p-4 rounded-lg border border-gray-200">
        <h4 className="text-sm font-medium text-gray-700 mb-3">Grant access to an employee:</h4>
        <div className="flex flex-col sm:flex-row gap-3">
          <select
            value={selectedEmployee}
            onChange={(e) => setSelectedEmployee(e.target.value)}
            className="flex-1 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
          >
            <option value="">Select an employee...</option>
            {employees.map(emp => {
              // Don't show if they already have an explicit record
              const hasAccess = accessList.find(a => a.employee_id === emp.id);
              if (hasAccess) return null;
              return (
                <option key={emp.id} value={emp.id}>
                  {emp.first_name} {emp.last_name} ({emp.employee_code})
                </option>
              );
            })}
          </select>
          
          <select
            value={selectedAccess}
            onChange={(e) => setSelectedAccess(e.target.value)}
            className="w-full sm:w-48 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
          >
            <option value="read">Read Only</option>
            <option value="write">Read & Write</option>
            <option value="hide">Hide Document</option>
          </select>

          <button
            onClick={handleGrantAccess}
            disabled={!selectedEmployee || setAccessMutation.isPending}
            className="bg-indigo-600 text-white px-4 py-2 rounded-md hover:bg-indigo-700 transition-colors disabled:opacity-50 whitespace-nowrap"
          >
            Save
          </button>
        </div>
      </div>

      <div>
        <h4 className="text-sm font-medium text-gray-700 mb-3">Employees with custom access:</h4>
        
        {accessList.length === 0 ? (
          <p className="text-sm text-gray-500 text-center py-4 border border-dashed rounded-lg">
            No custom access records. This document is hidden from all employees (except managers).
          </p>
        ) : (
          <div className="space-y-3">
            {accessList.map(access => {
              const emp = employees.find(e => e.id === access.employee_id);
              return (
                <div key={access.id} className="flex items-center justify-between p-3 bg-white border border-gray-200 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-700 font-bold">
                      {emp?.first_name?.[0] || '?'}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {emp ? `${emp.first_name} ${emp.last_name}` : 'Unknown Employee'}
                      </p>
                      <p className="text-xs text-gray-500">{emp?.employee_code}</p>
                    </div>
                  </div>
                  
                  <div className="flex items-center gap-2">
                    <select
                      value={access.access_level}
                      onChange={(e) => handleUpdateAccess(access.employee_id, e.target.value)}
                      disabled={setAccessMutation.isPending}
                      className={`text-sm rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 py-1.5 
                        ${access.access_level === 'write' ? 'bg-amber-50 text-amber-700' : 
                          access.access_level === 'read' ? 'bg-green-50 text-green-700' : 
                          'bg-red-50 text-red-700'}`}
                    >
                      <option value="read">Read Only</option>
                      <option value="write">Read & Write</option>
                      <option value="hide">Hidden</option>
                    </select>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
