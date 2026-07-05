import { useState } from 'react';
import { createPortal } from 'react-dom';
import { useMutation } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Key } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import api from '@/lib/api';

interface ChangePasswordModalProps {
  isOpen: boolean;
  onClose: () => void;
  employeeId: string;
  requireOldPassword: boolean;
}

export const ChangePasswordModal = ({ isOpen, onClose, employeeId, requireOldPassword }: ChangePasswordModalProps) => {
  const { t } = useTranslation();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: async () => {
      setError(null);
      if (newPassword !== confirmPassword) {
        throw new Error(t('auth.passwords_do_not_match'));
      }
      if (newPassword.length < 8) {
        throw new Error(t('auth.password_min_length'));
      }

      await api.put(`/employees/${employeeId}/password`, {
        old_password: requireOldPassword ? oldPassword : "",
        new_password: newPassword,
      });
    },
    onSuccess: () => {
      handleClose();
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to change password');
    },
  });

  const handleClose = () => {
    setOldPassword('');
    setNewPassword('');
    setConfirmPassword('');
    setError(null);
    onClose();
  };

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 sm:p-0">
      <div className="fixed inset-0 bg-background/80 backdrop-blur-sm" onClick={handleClose} />
      <Card className="relative w-full max-w-md shadow-lg border-border animate-in fade-in zoom-in-95 duration-200">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="w-5 h-5 text-primary" />
            {requireOldPassword ? t('auth.change_password') : t('auth.reset_employee_password')}
          </CardTitle>
          <CardDescription>
            {requireOldPassword 
              ? t('auth.enter_current_to_set_new')
              : t('auth.set_new_for_employee')}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-xl">
              {error}
            </div>
          )}

          {requireOldPassword && (
            <div className="space-y-2">
              <Label htmlFor="old_password">{t('auth.current_password')}</Label>
              <Input
                id="old_password"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder={t('auth.enter_current_password')}
              />
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="new_password">{t('auth.new_password')}</Label>
            <Input
              id="new_password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder={t('auth.minimum_8_chars')}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirm_password">{t('auth.confirm_new_password')}</Label>
            <Input
              id="confirm_password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder={t('auth.re_enter_new_password')}
            />
          </div>
        </CardContent>
        <CardFooter className="flex justify-end gap-2">
          <Button variant="outline" onClick={handleClose} disabled={mutation.isPending}>
            {t('auth.cancel')}
          </Button>
          <Button 
            onClick={() => mutation.mutate()} 
            disabled={mutation.isPending || !newPassword || !confirmPassword || (requireOldPassword && !oldPassword)}
          >
            {mutation.isPending ? t('auth.saving') : t('auth.save_password')}
          </Button>
        </CardFooter>
      </Card>
    </div>,
    document.body
  );
};
