import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Plus, Edit2, Trash2, Shield, Search } from 'lucide-react';
import { infoTableService } from '../../services/api/infoTableService';
import type { InfoTable, InfoTableRow } from '../../types/infoTable';
import { useAuthStore } from '@/store/authStore';
import RowModal from './RowModal';
import AccessModal from './AccessModal';
import CreateTableModal from './CreateTableModal';

const InfoTableView: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  
  const [table, setTable] = useState<InfoTable | null>(null);
  const [rows, setRows] = useState<InfoTableRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  
  const [isRowModalOpen, setIsRowModalOpen] = useState(false);
  const [editingRow, setEditingRow] = useState<InfoTableRow | null>(null);
  const [isAccessModalOpen, setIsAccessModalOpen] = useState(false);
  const [isEditTableModalOpen, setIsEditTableModalOpen] = useState(false);

  useEffect(() => {
    if (id) {
      fetchData();
    }
  }, [id]);

  const fetchData = async () => {
    try {
      setLoading(true);
      // Fetch table structure
      const tables = await infoTableService.getVisibleTables();
      const currentTable = tables.find(t => t.id === id);
      if (!currentTable) {
        navigate('/info-tables');
        return;
      }
      setTable(currentTable);

      // Fetch rows
      const tableRows = await infoTableService.getTableRows(id!);
      setRows(tableRows);
    } catch (error) {
      console.error('Failed to fetch table data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteRow = async (rowId: string) => {
    if (!window.confirm('Are you sure you want to delete this row?')) return;
    try {
      await infoTableService.deleteTableRow(id!, rowId);
      setRows(rows.filter(r => r.id !== rowId));
    } catch (error) {
      console.error('Failed to delete row:', error);
      alert('Failed to delete row. You might not have permission.');
    }
  };

  const handleDeleteTable = async () => {
    if (!window.confirm('Are you sure you want to delete this entire table? This action cannot be undone.')) return;
    try {
      await infoTableService.deleteTable(id!);
      navigate('/info-tables');
    } catch (error) {
      console.error('Failed to delete table:', error);
      alert('Failed to delete table. You might not have permission.');
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (!table) return null;

  // Filter rows based on search term
  const filteredRows = rows.filter(row => {
    if (!searchTerm) return true;
    return Object.values(row.data).some(val => 
      String(val).toLowerCase().includes(searchTerm.toLowerCase())
    );
  });

  const canManageAccess = table.my_access_level === 'manage' || user?.role === 'admin';
  const canWrite = table.my_access_level === 'manage' || table.my_access_level === 'write' || user?.role === 'admin'; 

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div className="flex items-center gap-4 text-sm text-gray-500 dark:text-gray-400 mb-2">
        <button 
          onClick={() => navigate('/info-tables')}
          className="hover:text-gray-900 dark:hover:text-white flex items-center transition-colors"
        >
          <ArrowLeft className="w-4 h-4 mr-1" /> Back to Hub
        </button>
      </div>

      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            {table.name}
          </h1>
          {table.description && (
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              {table.description}
            </p>
          )}
        </div>

        <div className="flex gap-3">
          {canManageAccess && (
            <>
              <button
                onClick={handleDeleteTable}
                className="px-4 py-2 bg-white dark:bg-gray-800 border border-red-300 dark:border-red-800/50 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors shadow-sm flex items-center gap-2"
              >
                <Trash2 className="w-4 h-4" />
                Delete
              </button>
              <button
                onClick={() => setIsEditTableModalOpen(true)}
                className="px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors shadow-sm flex items-center gap-2"
              >
                <Edit2 className="w-4 h-4" />
                Edit
              </button>
              <button
                onClick={() => setIsAccessModalOpen(true)}
                className="px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors shadow-sm flex items-center gap-2"
              >
                <Shield className="w-4 h-4" />
                Access
              </button>
            </>
          )}
          {canWrite && (
            <button
              onClick={() => {
                setEditingRow(null);
                setIsRowModalOpen(true);
              }}
              className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors shadow-sm flex items-center gap-2"
            >
              <Plus className="w-4 h-4" />
              Add Row
            </button>
          )}
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search in table..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-all text-sm"
            />
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left text-gray-500 dark:text-gray-400">
            <thead className="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-300">
              <tr>
                {table.columns.sort((a, b) => a.order - b.order).map(col => (
                  <th key={col.id} className="px-6 py-4 font-medium tracking-wider">
                    {col.name}
                  </th>
                ))}
                {canWrite && <th className="px-6 py-4 text-right">Actions</th>}
              </tr>
            </thead>
            <tbody>
              {filteredRows.length === 0 ? (
                <tr>
                  <td colSpan={table.columns.length + (canWrite ? 1 : 0)} className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
                    No rows found.
                  </td>
                </tr>
              ) : (
                filteredRows.map(row => (
                  <tr key={row.id} className="bg-white border-b dark:bg-gray-800 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                    {table.columns.sort((a, b) => a.order - b.order).map(col => (
                      <td key={col.id} className="px-6 py-4">
                        {col.type === 'link' && row.data[col.id] ? (
                          <a href={row.data[col.id]} target="_blank" rel="noreferrer" className="text-primary-600 hover:underline">
                            Link
                          </a>
                        ) : col.type === 'date' && row.data[col.id] ? (
                          new Date(row.data[col.id]).toLocaleDateString()
                        ) : (
                          row.data[col.id] || '-'
                        )}
                      </td>
                    ))}
                    {canWrite && (
                      <td className="px-6 py-4 text-right">
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => {
                              setEditingRow(row);
                              setIsRowModalOpen(true);
                            }}
                            className="p-1.5 text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30 rounded-md transition-colors"
                            title="Edit Row"
                          >
                            <Edit2 className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleDeleteRow(row.id)}
                            className="p-1.5 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30 rounded-md transition-colors"
                            title="Delete Row"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    )}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <RowModal
        isOpen={isRowModalOpen}
        onClose={() => {
          setIsRowModalOpen(false);
          setEditingRow(null);
        }}
        table={table}
        initialData={editingRow}
        onSave={(savedRow) => {
          if (editingRow) {
            setRows(rows.map(r => r.id === savedRow.id ? savedRow : r));
          } else {
            setRows([...rows, savedRow]);
          }
        }}
      />

      <AccessModal
        isOpen={isAccessModalOpen}
        onClose={() => setIsAccessModalOpen(false)}
        table={table}
      />

      <CreateTableModal
        isOpen={isEditTableModalOpen}
        onClose={() => setIsEditTableModalOpen(false)}
        initialData={table}
        onUpdated={(updatedTable) => {
          setTable({ ...table, ...updatedTable });
          setIsEditTableModalOpen(false);
        }}
      />
    </div>
  );
};

export default InfoTableView;
