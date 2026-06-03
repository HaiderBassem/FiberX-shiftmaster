import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { Login } from '../features/auth/Login';
import { DashboardLayout } from '../components/layout/DashboardLayout';
import { DepartmentList } from '../features/departments/DepartmentList';
import { EmployeeList } from '../features/employees/EmployeeList';
import { ShiftList } from '../features/shifts/ShiftList';
import { ScheduleView } from '../features/schedules/ScheduleView';
import { MyTasksWeekly } from '../features/tasks/TaskList';
import { TaskHub } from '../features/tasks/TaskHub';
import { RequestHub } from '../features/requests/RequestHub';
import { ApprovalDashboard } from '../features/approvals/ApprovalDashboard';
import { NotificationList } from '../features/notifications/NotificationList';
import { Dashboard } from '../features/dashboard/Dashboard';
import { DepartmentDetail } from '../features/departments/DepartmentDetail';
import { EmployeeDetail } from '../features/employees/EmployeeDetail';
import InfoTableHub from '../features/infotables/InfoTableHub';
import InfoTableView from '../features/infotables/InfoTableView';
import { HelpDocumentList } from '../features/help/HelpDocumentList';
import { HelpDocumentView } from '../features/help/HelpDocumentView';
import { HelpDocumentEditor } from '../features/help/HelpDocumentEditor';
import { AnnouncementManager } from '../features/announcements/AnnouncementManager';
import InteractiveCalendar from '../features/calendar/InteractiveCalendar';

const ProtectedRoute = ({ children, allowedRoles, allowHelpDocsAccess, allowAnnouncementsAccess }: { children: React.ReactNode, allowedRoles?: string[], allowHelpDocsAccess?: boolean, allowAnnouncementsAccess?: boolean }) => {
  const { isAuthenticated, user } = useAuthStore();
  
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  
  if (allowedRoles && user && !allowedRoles.includes(user.role)) {
    if (allowHelpDocsAccess && (user as any).can_manage_help_docs) {
      // allow
    } else if (allowAnnouncementsAccess && (user as any).can_post_announcements) {
      // allow
    } else {
      return <Navigate to="/unauthorized" replace />;
    }
  }
  
  return <>{children}</>;
};

export const AppRoutes = () => {
  return (
    <Router>
      <Routes>
        <Route path="/login" element={<Login />} />
        
        <Route path="/" element={<ProtectedRoute><DashboardLayout /></ProtectedRoute>}>
          <Route index element={<Dashboard />} />

          {/* ── Employee self-service routes (all roles) ── */}
          <Route 
            path="calendar" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <InteractiveCalendar />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="tasks" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <MyTasksWeekly />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="requests" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <RequestHub />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="notifications" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <NotificationList />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="info-tables" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <InfoTableHub />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="info-tables/:id" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <InfoTableView />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="help" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <HelpDocumentList />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="help/new" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']} allowHelpDocsAccess={true}>
                <HelpDocumentEditor />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="help/:id" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <HelpDocumentView />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="help/:id/edit" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']} allowHelpDocsAccess={true}>
                <HelpDocumentEditor />
              </ProtectedRoute>
            } 
          />

          {/* ── Supervisor routes (team_leader, manager, admin) ── */}
          <Route 
            path="approvals" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']}>
                <ApprovalDashboard />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="task-management" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']}>
                <TaskHub />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="shifts" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']}>
                <div className="space-y-12">
                  <ScheduleView />
                  <ShiftList />
                </div>
              </ProtectedRoute>
            } 
          />

          {/* ── Admin / Manager / Approver routes ── */}
          <Route 
            path="announcements/manage" 
            element={
              <ProtectedRoute allowedRoles={['manager', 'admin']} allowAnnouncementsAccess={true}>
                <AnnouncementManager />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="departments" 
            element={
              <ProtectedRoute allowedRoles={['admin']}>
                <DepartmentList />
              </ProtectedRoute>
            } 
          />

          <Route
            path="departments/:id"
            element={
              <ProtectedRoute allowedRoles={['admin']}>
                <DepartmentDetail />
              </ProtectedRoute>
            }
          />
          <Route 
            path="employees" 
            element={
              <ProtectedRoute allowedRoles={['admin', 'manager', 'team_leader']}>
                <EmployeeList />
              </ProtectedRoute>
            } 
          />
          <Route
            path="employees/:id"
            element={
              <ProtectedRoute allowedRoles={['admin', 'manager', 'team_leader']}>
                <EmployeeDetail />
              </ProtectedRoute>
            }
          />
        </Route>
        
        <Route path="/unauthorized" element={
          <div className="min-h-screen bg-background flex items-center justify-center">
            <div className="text-center">
              <h1 className="text-4xl font-bold text-destructive mb-4">Unauthorized</h1>
              <p className="text-muted-foreground">You don't have permission to access this page.</p>
            </div>
          </div>
        } />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  );
};
