import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Table as TableIcon, 
  Plus, 
  Search, 
  MoreVertical, 
  FileText,
  Lock,
  Globe,
  Shield
} from 'lucide-react';
import { infoTableService } from '../../services/api/infoTableService';
import type { InfoTable } from '../../types/infoTable';
import { useAuthStore } from '@/store/authStore';
import CreateTableModal from './CreateTableModal';
import { TablePermissionsModal } from './TablePermissionsModal';
import { DraggableGrid, GridLayout } from '../../components/ui/DraggableGrid';
import api from '@/lib/api';

const InfoTableHub: React.FC = () => {
  const [tables, setTables] = useState<InfoTable[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isPermissionsModalOpen, setIsPermissionsModalOpen] = useState(false);
  const { user, updateUserPreferences } = useAuthStore();
  const navigate = useNavigate();
  const [layout, setLayout] = useState<GridLayout>({ folders: {}, order: [] });

  useEffect(() => {
    if (user?.ui_preferences?.reference_layout) {
      setLayout(user.ui_preferences.reference_layout);
    }
  }, [user]);

  const handleLayoutChange = async (newLayout: GridLayout) => {
    setLayout(newLayout);
    const newPrefs = { reference_layout: newLayout };
    updateUserPreferences(newPrefs);
    if (user?.id) {
      try {
        await api.put(`/employees/${user.id}/preferences`, { ui_preferences: { ...user?.ui_preferences, ...newPrefs } });
      } catch (e) {
        console.error("Failed to save layout", e);
      }
    }
  };

  const fetchTables = async () => {
    try {
      setLoading(true);
      const data = await infoTableService.getVisibleTables();
      setTables(data);
    } catch (error) {
      console.error('Failed to fetch info tables:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTables();
  }, []);

  const filteredTables = tables.filter(t => 
    t.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
    (t.description?.toLowerCase() || '').includes(searchTerm.toLowerCase())
  );

  // can create table?
  // Only admin, manager, team_leader or employee with can_create_tables
  const canCreate = ['admin', 'manager', 'team_leader'].includes(user?.role || '') || user?.can_create_tables;
  const canManagePermissions = user?.role === 'manager' || user?.role === 'admin';

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
            <TableIcon className="w-6 h-6 text-primary-500" />
            Reference
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Dynamic data tables for assets, contacts, and custom records
          </p>
        </div>
        
        <div className="flex gap-2">
          {canManagePermissions && (
            <button
              onClick={() => setIsPermissionsModalOpen(true)}
              className="flex items-center gap-2 bg-white text-gray-700 border border-gray-300 px-4 py-2 rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
            >
              <Shield className="w-4 h-4" />
              <span className="hidden sm:inline">Manage Permissions</span>
            </button>
          )}
          {canCreate && (
            <button
              onClick={() => setIsCreateModalOpen(true)}
              className="flex items-center gap-2 bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition-colors shadow-sm"
            >
              <Plus className="w-4 h-4" />
              Create Table
            </button>
          )}
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-4">
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search tables..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-all"
          />
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : filteredTables.length === 0 ? (
        <div className="text-center py-12 bg-white dark:bg-gray-800 rounded-xl border border-dashed border-gray-300 dark:border-gray-700">
          <TableIcon className="w-12 h-12 text-gray-400 mx-auto mb-3" />
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">No tables found</h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {searchTerm ? "Try adjusting your search" : "Get started by creating a new table"}
          </p>
        </div>
      ) : (
        <DraggableGrid
          items={filteredTables}
          layout={layout}
          onLayoutChange={handleLayoutChange}
          isSearchActive={searchTerm.length > 0}
          onItemClick={(table) => navigate(`/info-tables/${table.id}`)}
          renderItem={(table) => (
            <div 
              className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-5 hover:shadow-md hover:border-primary-300 dark:hover:border-primary-700 transition-all cursor-pointer group flex flex-col h-full"
            >
              <div className="flex justify-between items-start mb-4">
                <div className="bg-primary-50 dark:bg-primary-900/30 p-3 rounded-lg text-primary-600 dark:text-primary-400">
                  <FileText className="w-6 h-6" />
                </div>
                <button 
                  onClick={(e) => {
                    e.stopPropagation();
                  }}
                  className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <MoreVertical className="w-4 h-4" />
                </button>
              </div>
              
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-1">
                {table.name}
              </h3>
              
              <p className="text-sm text-gray-500 dark:text-gray-400 line-clamp-2 mb-4 flex-grow">
                {table.description || "No description provided."}
              </p>
              
              <div className="flex items-center justify-between mt-auto pt-4 border-t border-gray-100 dark:border-gray-700">
                <div className="flex items-center text-xs text-gray-500 dark:text-gray-400">
                  <span className="font-medium bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded-md">
                    {table.columns.length} columns
                  </span>
                </div>
                
                <div className="flex items-center text-xs text-gray-500 dark:text-gray-400 gap-1">
                  {table.department_id ? (
                    <><Lock className="w-3 h-3" /> Dept Restricted</>
                  ) : (
                    <><Globe className="w-3 h-3" /> Global</>
                  )}
                </div>
              </div>
            </div>
          )}
        />
      )}

      <CreateTableModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onCreated={(newTable) => {
          setTables([newTable, ...tables]);
          setIsCreateModalOpen(false);
        }}
      />
      
      <TablePermissionsModal
        isOpen={isPermissionsModalOpen}
        onClose={() => setIsPermissionsModalOpen(false)}
      />
    </div>
  );
};

export default InfoTableHub;
