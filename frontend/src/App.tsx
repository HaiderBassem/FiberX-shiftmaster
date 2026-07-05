import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { AppRoutes } from './routes/AppRoutes';
import { ThemeProvider } from './components/ThemeProvider';
import { Toaster } from 'sonner';
import { NotificationProvider } from './providers/NotificationProvider';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function App() {
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <NotificationProvider>
          <AppRoutes />
          <Toaster richColors position="top-right" />
        </NotificationProvider>
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ThemeProvider>
  );
}

export default App;
