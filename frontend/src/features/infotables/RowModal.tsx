import React, { useState, useEffect } from 'react';
import { X } from 'lucide-react';
import { infoTableService } from '../../services/api/infoTableService';
import type { InfoTable, InfoTableRow } from '../../types/infoTable';

interface RowModalProps {
  isOpen: boolean;
  onClose: () => void;
  table: InfoTable;
  initialData: InfoTableRow | null;
  onSave: (row: InfoTableRow) => void;
}

const RowModal: React.FC<RowModalProps> = ({ isOpen, onClose, table, initialData, onSave }) => {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialData) {
      setFormData(initialData.data || {});
    } else {
      setFormData({});
    }
  }, [initialData, isOpen]);

  if (!isOpen) return null;

  const handleChange = (colId: string, value: any) => {
    setFormData(prev => ({ ...prev, [colId]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      setError(null);
      
      let savedRow;
      if (initialData) {
        savedRow = await infoTableService.updateTableRow(table.id, initialData.id, formData);
      } else {
        savedRow = await infoTableService.createTableRow(table.id, formData);
      }
      onSave(savedRow);
      onClose();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to save row');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-gray-900/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg overflow-hidden">
        <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            {initialData ? 'Edit Row' : 'Add Row'}
          </h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6">
          {error && (
            <div className="mb-6 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-2">
            {table.columns.sort((a, b) => a.order - b.order).map(col => (
              <div key={col.id}>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {col.name}
                </label>
                {col.type === 'text' || col.type === 'link' ? (
                  <input
                    type="text"
                    value={formData[col.id] || ''}
                    onChange={(e) => handleChange(col.id, e.target.value)}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                  />
                ) : col.type === 'number' ? (
                  <input
                    type="number"
                    value={formData[col.id] || ''}
                    onChange={(e) => handleChange(col.id, Number(e.target.value))}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                  />
                ) : col.type === 'date' ? (
                  <input
                    type="date"
                    value={formData[col.id] || ''}
                    onChange={(e) => handleChange(col.id, e.target.value)}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                  />
                ) : (
                  <input
                    type="text"
                    value={formData[col.id] || ''}
                    onChange={(e) => handleChange(col.id, e.target.value)}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                  />
                )}
              </div>
            ))}
          </div>

          <div className="mt-8 flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg font-medium transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-medium transition-colors disabled:opacity-70 flex items-center"
            >
              {loading ? (
                <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin mr-2" />
              ) : null}
              {initialData ? 'Save Changes' : 'Add Row'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default RowModal;
