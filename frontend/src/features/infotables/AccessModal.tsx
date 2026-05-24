import React, { useState, useEffect } from 'react';
import { X, Users, UserPlus } from 'lucide-react';
import { infoTableService } from '../../services/api/infoTableService';
import { departmentService } from '../../services/api/departmentService';
import { employeeService } from '../../services/api/employeeService';
import { InfoTable, InfoTableDepartmentAccess, InfoTableEmployeeAccess } from '../../types/infoTable';
import { Department } from '../../types/department';
import { Employee } from '../../types/employee';

interface AccessModalProps {
  isOpen: boolean;
  onClose: () => void;
  table: InfoTable;
}

const AccessModal: React.FC<AccessModalProps> = ({ isOpen, onClose, table }) => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [depAccesses, setDepAccesses] = useState<InfoTableDepartmentAccess[]>([]);
  const [empAccesses, setEmpAccesses] = useState<InfoTableEmployeeAccess[]>([]);
  const [loading, setLoading] = useState(true);

  // Form states
  const [selectedDepId, setSelectedDepId] = useState('');
  const [selectedEmpId, setSelectedEmpId] = useState('');
  const [accessLevel, setAccessLevel] = useState('read');

  useEffect(() => {
    if (isOpen) {
      fetchData();
    }
  }, [isOpen]);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [deps, emps, access] = await Promise.all([
        departmentService.getAll(),
        employeeService.getAll(),
        infoTableService.getAccessLists(table.id)
      ]);
      setDepartments(deps);
      setEmployees(emps);
      setDepAccesses(access.departments || []);
      setEmpAccesses(access.employees || []);
    } catch (error) {
      console.error('Failed to load access data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleShareWithDepartment = async () => {
    if (!selectedDepId) return;
    try {
      await infoTableService.shareWithDepartment(table.id, selectedDepId);
      setSelectedDepId('');
      fetchData(); // Refresh list
    } catch (error: any) {
      alert(error.response?.data?.error || 'Failed to share with department');
    }
  };

  const handleAddEmployeeAccess = async () => {
    if (!selectedEmpId) return;
    try {
      await infoTableService.addEmployeeAccess(table.id, selectedEmpId, accessLevel);
      setSelectedEmpId('');
      fetchData(); // Refresh list
    } catch (error: any) {
      alert(error.response?.data?.error || 'Failed to add employee access');
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-gray-900/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-2xl overflow-hidden">
        <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <Users className="w-5 h-5 text-primary-500" /> Manage Access
          </h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6">
          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : (
            <div className="space-y-8">
              {/* Department Sharing */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-3 border-b border-gray-200 dark:border-gray-700 pb-2">
                  Share with Department
                </h3>
                <div className="flex gap-3 mb-4">
                  <select
                    value={selectedDepId}
                    onChange={(e) => setSelectedDepId(e.target.value)}
                    className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="">Select Department...</option>
                    {departments.map(d => (
                      <option key={d.id} value={d.id}>{d.name}</option>
                    ))}
                  </select>
                  <button
                    onClick={handleShareWithDepartment}
                    disabled={!selectedDepId}
                    className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 transition-colors"
                  >
                    Share
                  </button>
                </div>
                
                {depAccesses.length > 0 && (
                  <div className="bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700 p-3">
                    <ul className="space-y-2">
                      {depAccesses.map(da => {
                        const dept = departments.find(d => d.id === da.department_id);
                        return (
                          <li key={da.id} className="flex justify-between items-center text-sm text-gray-700 dark:text-gray-300">
                            <span>{dept?.name || da.department_id}</span>
                            <span className="text-xs bg-gray-200 dark:bg-gray-700 px-2 py-1 rounded text-gray-600 dark:text-gray-400">Granted</span>
                          </li>
                        );
                      })}
                    </ul>
                  </div>
                )}
              </div>

              {/* Employee Access */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-3 border-b border-gray-200 dark:border-gray-700 pb-2 flex items-center gap-2">
                  <UserPlus className="w-4 h-4 text-gray-500" /> Employee Exceptions
                </h3>
                <div className="flex gap-3 mb-4">
                  <select
                    value={selectedEmpId}
                    onChange={(e) => setSelectedEmpId(e.target.value)}
                    className="flex-[2] px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="">Select Employee...</option>
                    {employees.map(e => (
                      <option key={e.id} value={e.id}>{e.first_name} {e.last_name} ({e.email})</option>
                    ))}
                  </select>
                  <select
                    value={accessLevel}
                    onChange={(e) => setAccessLevel(e.target.value)}
                    className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="read">Read Only</option>
                    <option value="write">Read & Write</option>
                  </select>
                  <button
                    onClick={handleAddEmployeeAccess}
                    disabled={!selectedEmpId}
                    className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 transition-colors"
                  >
                    Add
                  </button>
                </div>
                
                {empAccesses.length > 0 && (
                  <div className="bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700 p-3">
                    <ul className="space-y-2">
                      {empAccesses.map(ea => {
                        const emp = employees.find(e => e.id === ea.employee_id);
                        return (
                          <li key={ea.id} className="flex justify-between items-center text-sm text-gray-700 dark:text-gray-300">
                            <span>{emp ? `${emp.first_name} ${emp.last_name}` : ea.employee_id}</span>
                            <span className={`text-xs px-2 py-1 rounded ${ea.access_level === 'write' ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300' : 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400'}`}>
                              {ea.access_level.toUpperCase()}
                            </span>
                          </li>
                        );
                      })}
                    </ul>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AccessModal;
