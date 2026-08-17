import { lazy, Suspense } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '../store/authStore';
import { Login } from '../features/auth/Login';
import { DashboardLayout } from '../components/layout/DashboardLayout';
import { Dashboard } from '../features/dashboard/Dashboard';

// Route-level code splitting. Every feature used to be imported eagerly, so a
// user landing on the login page downloaded the entire application — around
// 2.3MB of JavaScript — before seeing anything. Login, the shell and the
// dashboard stay eager because they are on the path to first paint; the rest
// load when their route is first visited.
const AnnouncementManager = lazy(() => import('../features/announcements/AnnouncementManager').then(m => ({ default: m.AnnouncementManager })));
const ApprovalDashboard = lazy(() => import('../features/approvals/ApprovalDashboard').then(m => ({ default: m.ApprovalDashboard })));
const DepartmentDetail = lazy(() => import('../features/departments/DepartmentDetail').then(m => ({ default: m.DepartmentDetail })));
const DepartmentList = lazy(() => import('../features/departments/DepartmentList').then(m => ({ default: m.DepartmentList })));
const EmployeeDetail = lazy(() => import('../features/employees/EmployeeDetail').then(m => ({ default: m.EmployeeDetail })));
const EmployeeList = lazy(() => import('../features/employees/EmployeeList').then(m => ({ default: m.EmployeeList })));
const ExternalToolsList = lazy(() => import('../features/external-tools/ExternalToolsList').then(m => ({ default: m.ExternalToolsList })));
const FiberxDataEditor = lazy(() => import('../features/fiberx-data/FiberxDataEditor').then(m => ({ default: m.FiberxDataEditor })));
const FiberxDataHub = lazy(() => import('../features/fiberx-data/FiberxDataHub').then(m => ({ default: m.FiberxDataHub })));
const FiberxDataView = lazy(() => import('../features/fiberx-data/FiberxDataView').then(m => ({ default: m.FiberxDataView })));
const HandoverBoard = lazy(() => import('../features/handovers/HandoverBoard'));
const HelpDocumentEditor = lazy(() => import('../features/help/HelpDocumentEditor').then(m => ({ default: m.HelpDocumentEditor })));
const HelpDocumentList = lazy(() => import('../features/help/HelpDocumentList').then(m => ({ default: m.HelpDocumentList })));
const HelpDocumentView = lazy(() => import('../features/help/HelpDocumentView').then(m => ({ default: m.HelpDocumentView })));
const InboxPage = lazy(() => import('../features/notifications/InboxPage').then(m => ({ default: m.InboxPage })));
const InfoTableHub = lazy(() => import('../features/infotables/InfoTableHub'));
const InfoTableView = lazy(() => import('../features/infotables/InfoTableView'));
const InteractiveCalendar = lazy(() => import('../features/calendar/InteractiveCalendar'));
const LeaveTypeManager = lazy(() => import('../features/leaves/LeaveTypeManager').then(m => ({ default: m.LeaveTypeManager })));
const ModuleAccessSettings = lazy(() => import('../features/settings/ModuleAccessSettings').then(m => ({ default: m.ModuleAccessSettings })));
const MyTasksWeekly = lazy(() => import('../features/tasks/TaskList').then(m => ({ default: m.MyTasksWeekly })));
const RequestHub = lazy(() => import('../features/requests/RequestHub').then(m => ({ default: m.RequestHub })));
const ScheduleView = lazy(() => import('../features/schedules/ScheduleView').then(m => ({ default: m.ScheduleView })));
const ServiceHub = lazy(() => import('../features/services/ServiceHub').then(m => ({ default: m.ServiceHub })));
const ShiftList = lazy(() => import('../features/shifts/ShiftList').then(m => ({ default: m.ShiftList })));
const TaskHub = lazy(() => import('../features/tasks/TaskHub').then(m => ({ default: m.TaskHub })));
const TicketList = lazy(() => import('../features/tickets/TicketList').then(m => ({ default: m.TicketList })));
const UserProfile = lazy(() => import('../features/employees/UserProfile').then(m => ({ default: m.UserProfile })));

// Every user who hits a route they may not open sees this page, so it is one of
// the more visible surfaces that was still hardcoded English.
const Unauthorized = () => {
  const { t } = useTranslation();
  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold text-destructive mb-4">{t('common.unauthorized_title')}</h1>
        <p className="text-muted-foreground">{t('common.unauthorized_message')}</p>
      </div>
    </div>
  );
};

// Shown while a route chunk is being fetched.
const RouteFallback = () => (
  <div className="flex items-center justify-center py-24" role="status" aria-live="polite">
    <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
    <span className="sr-only">Loading</span>
  </div>
);

const ProtectedRoute = ({ children, allowedRoles, allowHelpDocsAccess, allowAnnouncementsAccess }: { children: React.ReactNode, allowedRoles?: string[], allowHelpDocsAccess?: boolean, allowAnnouncementsAccess?: boolean }) => {
  const { isAuthenticated, user } = useAuthStore();
  
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  
  if (allowedRoles && user && !allowedRoles.includes(user.role)) {
    if (allowHelpDocsAccess && user.can_manage_help_docs) {
      // allow
    } else if (allowAnnouncementsAccess && user.can_post_announcements) {
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
      <Suspense fallback={<RouteFallback />}>
      <Routes>
        <Route path="/login" element={<Login />} />
        
        <Route path="/" element={<ProtectedRoute><DashboardLayout /></ProtectedRoute>}>
          <Route index element={<Dashboard />} />
          <Route path="profile" element={<UserProfile />} />

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
                <InboxPage />
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
            path="external-tools" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <ExternalToolsList />
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
              <ProtectedRoute allowedRoles={['admin', 'manager']} allowHelpDocsAccess>
                <HelpDocumentEditor />
              </ProtectedRoute>
            }
          />
          <Route 
            path="fiberx-data/new" 
            element={
              <ProtectedRoute allowedRoles={['admin', 'manager', 'team_leader', 'employee']}>
                <FiberxDataEditor />
              </ProtectedRoute>
            }
          />
          <Route 
            path="fiberx-data/:id/edit" 
            element={
              <ProtectedRoute allowedRoles={['admin', 'manager', 'team_leader', 'employee']}>
                <FiberxDataEditor />
              </ProtectedRoute>
            }
          />
          
          {/* ── Management & Supervisor routes ── */}
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

          {/* Handovers */}
          <Route
            path="handovers"
            element={
              <ProtectedRoute>
                <HandoverBoard />
              </ProtectedRoute>
            }
          />
          <Route
            path="tickets"
            element={
              <ProtectedRoute>
                <TicketList />
              </ProtectedRoute>
            }
          />
          <Route
            path="services"
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <ServiceHub />
              </ProtectedRoute>
            }
          />
          <Route 
            path="fiberx-data" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <FiberxDataHub />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="fiberx-data/:id" 
            element={
              <ProtectedRoute allowedRoles={['employee', 'team_leader', 'manager', 'admin']}>
                <FiberxDataView />
              </ProtectedRoute>
            } 
          />
          
          {/* ── Help / Info Bank Editor (role restricted) ── */}
          <Route 
            path="announcements/manage" 
            element={
              <ProtectedRoute allowedRoles={['manager', 'admin']} allowAnnouncementsAccess={true}>
                <AnnouncementManager />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="module-settings" 
            element={
              <ProtectedRoute allowedRoles={['team_leader', 'manager', 'admin']}>
                <ModuleAccessSettings />
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
            path="leave-config" 
            element={
              <ProtectedRoute allowedRoles={['admin']}>
                <LeaveTypeManager />
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
        
        <Route path="/unauthorized" element={<Unauthorized />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      </Suspense>
    </Router>
  );
};
