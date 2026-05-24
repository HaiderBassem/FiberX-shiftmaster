import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { Login } from '../features/auth/Login';
import { DashboardLayout } from '../components/layout/DashboardLayout';
import { DepartmentList } from '../features/departments/DepartmentList';
import { EmployeeList } from '../features/employees/EmployeeList';
import { ShiftList } from '../features/shifts/ShiftList';
import { ScheduleView } from '../features/schedules/ScheduleView';
import { MyTasksWeekly } from '../features/tasks/TaskList';
import { TaskManagement } from '../features/tasks/TaskManagement';
import { TaskBoards } from '../features/tasks/TaskBoards';
import { TaskHistory } from '../features/tasks/TaskHistory';
import { LeaveList } from '../features/leaves/LeaveList';
import { SwapList } from '../features/swaps/SwapList';
import { ApprovalDashboard } from '../features/approvals/ApprovalDashboard';
import { NotificationList } from '../features/notifications/NotificationList';
import { Dashboard } from '../features/dashboard/Dashboard';
import { DepartmentDetail } from '../features/departments/DepartmentDetail';
import { EmployeeDetail } from '../features/employees/EmployeeDetail';
import InfoTableHub from '../features/infotables/InfoTableHub';
import InfoTableView from '../features/infotables/InfoTableView';

const ProtectedRoute = ({ children, allowedRoles }: { children: React.ReactNode, allowedRoles?: string[] }) => {
  const { isAuthenticated, user } = useAuthStore();
  
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  
  if (allowedRoles && user && !allowedRoles.includes(user.role)) {
    return <Navigate to="/unauthorized" replace />;
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
            path="tasks" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <MyTasksWeekly />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="leaves" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <LeaveList />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="swaps" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <SwapList />
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
                <TaskManagement />
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
          <Route 
            path="task-boards" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']}>
                <TaskBoards />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="task-history" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']}>
                <TaskHistory />
              </ProtectedRoute>
            } 
          />

          {/* ── Admin routes (admin only) ── */}
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
