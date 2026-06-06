import React, { useState, useEffect } from 'react';
import { X, Plus, Trash2, GripVertical } from 'lucide-react';
import { infoTableService } from '../../services/api/infoTableService';
import type { InfoTable, InfoTableColumn } from '../../types/infoTable';

interface CreateTableModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreated?: (table: InfoTable) => void;
  onUpdated?: (table: InfoTable) => void;
  initialData?: InfoTable | null;
}

const CreateTableModal: React.FC<CreateTableModalProps> = ({ isOpen, onClose, onCreated, onUpdated, initialData }) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [columns, setColumns] = useState<InfoTableColumn[]>([
    { id: 'col_1', name: 'Field 1', type: 'text', order: 1 }
  ]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      if (initialData) {
        setName(initialData.name);
        setDescription(initialData.description || '');
        setColumns(initialData.columns && initialData.columns.length > 0 ? initialData.columns : [{ id: 'col_1', name: 'Field 1', type: 'text', order: 1 }]);
      } else {
        setName('');
        setDescription('');
        setColumns([{ id: 'col_1', name: 'Field 1', type: 'text', order: 1 }]);
      }
      setError(null);
    }
  }, [isOpen, initialData]);

  if (!isOpen) return null;

  const handleAddColumn = () => {
    const newId = `col_${Date.now()}`;
    setColumns([
      ...columns,
      { id: newId, name: `Field ${columns.length + 1}`, type: 'text', order: columns.length + 1 }
    ]);
  };

  const handleRemoveColumn = (id: string) => {
    setColumns(columns.filter(c => c.id !== id));
  };

  const handleChangeColumn = (id: string, field: keyof InfoTableColumn, value: string) => {
    setColumns(columns.map(c => c.id === id ? { ...c, [field]: value } : c));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Table name is required');
      return;
    }
    if (columns.length === 0) {
      setError('At least one column is required');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      if (initialData && onUpdated) {
        const updatedTable = await infoTableService.updateTable(initialData.id, {
          name,
          description,
          columns: columns.map((c, idx) => ({ ...c, order: idx + 1 }))
        });
        onUpdated(updatedTable);
      } else if (onCreated) {
        const newTable = await infoTableService.createTable({
          name,
          description,
          columns: columns.map((c, idx) => ({ ...c, order: idx + 1 }))
        });
        onCreated(newTable);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || `Failed to ${initialData ? 'update' : 'create'} table`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-gray-900/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-2xl overflow-hidden">
        <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            {initialData ? 'Edit Table' : 'Create New Table'}
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

          <div className="space-y-4 mb-8">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Table Name
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Employee Assets"
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500"
                required
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Description (Optional)
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What is this table for?"
                rows={2}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 resize-none"
              />
            </div>
          </div>

          <div className="mb-4 flex justify-between items-center">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Columns configuration
            </label>
            <button
              type="button"
              onClick={handleAddColumn}
              className="text-sm flex items-center text-primary-600 hover:text-primary-700 font-medium"
            >
              <Plus className="w-4 h-4 mr-1" /> Add Column
            </button>
          </div>

          <div className="space-y-3 max-h-[40vh] overflow-y-auto pr-2">
            {columns.map((col) => (
              <div key={col.id} className="flex gap-3 items-center bg-gray-50 dark:bg-gray-900/50 p-3 rounded-lg border border-gray-200 dark:border-gray-700">
                <GripVertical className="w-4 h-4 text-gray-400 cursor-move flex-shrink-0" />
                
                <div className="flex-1">
                  <input
                    type="text"
                    value={col.name}
                    onChange={(e) => handleChangeColumn(col.id, 'name', e.target.value)}
                    placeholder="Column Name"
                    className="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                    required
                  />
                </div>
                
                <div className="w-32">
                  <select
                    value={col.type}
                    onChange={(e) => handleChangeColumn(col.id, 'type', e.target.value)}
                    className="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  >
                    <option value="text">Text</option>
                    <option value="number">Number</option>
                    <option value="date">Date</option>
                    <option value="link">Link</option>
                  </select>
                </div>

                <button
                  type="button"
                  onClick={() => handleRemoveColumn(col.id)}
                  disabled={columns.length === 1}
                  className="p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30 rounded disabled:opacity-50"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
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
              {initialData ? 'Update Table' : 'Create Table'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateTableModal;
