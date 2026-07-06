import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Send, MoonStar, Shuffle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';

export type SwapMode = 'off' | 'shift';

interface SwapRequestModalProps {
  isOpen: boolean;
  onClose: () => void;
  targetEmployeeId: string;
  setTargetEmployeeId: (id: string) => void;
  shiftDate: string;
  setShiftDate: (date: string) => void;
  reason: string;
  setReason: (reason: string) => void;
  onSubmit: () => void;
  isPending: boolean;
  canSubmit: boolean;
  error: string | null;
}

export const SwapRequestModal: React.FC<SwapRequestModalProps> = ({
  isOpen, onClose, targetEmployeeId, setTargetEmployeeId,
  shiftDate, setShiftDate, reason, setReason, onSubmit, isPending, canSubmit, error
}) => {
  const { t } = useTranslation();
  const [swapMode, setSwapMode] = useState<SwapMode>('off');

  // Reset target when mode or date changes
  useEffect(() => {
    setTargetEmployeeId('');
  }, [swapMode, shiftDate]);

  // Fetch eligible employees based on mode
  const { data: eligibleEmployees, isLoading: loadingEmployees } = useQuery({
    queryKey: ['swap-eligible', swapMode, shiftDate],
    queryFn: async () => {
      if (!shiftDate) return [];
      const endpoint = swapMode === 'off'
        ? `/swaps/eligible-targets?date=${shiftDate}`
        : `/swaps/eligible-shift-targets?date=${shiftDate}`;
      const res = await api.get(endpoint);
      return res.data?.data || [];
    },
    enabled: isOpen && !!shiftDate,
  });

  const selectClass = "w-full h-11 px-4 py-2 rounded-xl bg-muted/50 border-transparent focus:bg-background focus:border-primary text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all";
  const inputClass = "w-full h-11 px-4 py-2 rounded-xl bg-muted/50 border-transparent focus:bg-background focus:border-primary text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all";

  const employees = eligibleEmployees || [];

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50"
          />
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ type: "spring", duration: 0.5, bounce: 0.3 }}
            className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[90%] max-w-md bg-background rounded-3xl shadow-2xl z-50 overflow-hidden border border-white/10"
          >
            <div className="p-6 space-y-5">
              {/* Header */}
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-bold">{t('swaps.request_shift_swap')}</h2>
                <button onClick={onClose} className="p-2 rounded-full hover:bg-muted transition-colors">
                  <X className="w-5 h-5 text-muted-foreground" />
                </button>
              </div>

              {error && (
                <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm font-medium">
                  {error}
                </div>
              )}

              {/* Swap Mode Selector */}
              <div className="space-y-2">
                <Label className="text-muted-foreground ml-1">{t('swaps.swap_mode', 'نوع الـ Swap')}</Label>
                <div className="grid grid-cols-2 gap-2">
                  {/* Off Mode */}
                  <label
                    htmlFor="mode-off"
                    className={`flex items-center gap-3 p-3 rounded-xl border-2 cursor-pointer transition-all select-none ${
                      swapMode === 'off'
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border bg-muted/30 text-muted-foreground hover:border-primary/50'
                    }`}
                  >
                    <input
                      type="radio"
                      id="mode-off"
                      name="swapMode"
                      value="off"
                      checked={swapMode === 'off'}
                      onChange={() => setSwapMode('off')}
                      className="sr-only"
                    />
                    <MoonStar className="w-4 h-4 shrink-0" />
                    <div>
                      <p className="text-sm font-semibold">{t('swaps.mode_off', 'استراحة (Off)')}</p>
                      <p className="text-xs opacity-70">{t('swaps.mode_off_desc', 'موظفين عندهم Off')}</p>
                    </div>
                  </label>

                  {/* Shift Mode */}
                  <label
                    htmlFor="mode-shift"
                    className={`flex items-center gap-3 p-3 rounded-xl border-2 cursor-pointer transition-all select-none ${
                      swapMode === 'shift'
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border bg-muted/30 text-muted-foreground hover:border-primary/50'
                    }`}
                  >
                    <input
                      type="radio"
                      id="mode-shift"
                      name="swapMode"
                      value="shift"
                      checked={swapMode === 'shift'}
                      onChange={() => setSwapMode('shift')}
                      className="sr-only"
                    />
                    <Shuffle className="w-4 h-4 shrink-0" />
                    <div>
                      <p className="text-sm font-semibold">{t('swaps.mode_shift', 'تبادل شفت')}</p>
                      <p className="text-xs opacity-70">{t('swaps.mode_shift_desc', 'موظفين بشفت ثاني')}</p>
                    </div>
                  </label>
                </div>
              </div>

              <div className="space-y-4">
                {/* Date Picker */}
                <div className="space-y-2">
                  <Label className="text-muted-foreground ml-1">{t('swaps.your_shift_date')}</Label>
                  <input
                    type="date"
                    className={inputClass}
                    value={shiftDate}
                    onChange={(e) => setShiftDate(e.target.value)}
                  />
                </div>

                {/* Employee Selector */}
                <div className="space-y-2">
                  <Label className="text-muted-foreground ml-1">
                    {swapMode === 'off'
                      ? t('swaps.select_off_colleague', 'اختر موظف عنده Off')
                      : t('swaps.select_shift_colleague', 'اختر موظف بشفت ثاني')}
                  </Label>

                  {loadingEmployees ? (
                    <div className="h-11 rounded-xl bg-muted/50 animate-pulse" />
                  ) : employees.length === 0 ? (
                    <div className="h-11 flex items-center px-4 rounded-xl bg-muted/30 border border-border text-muted-foreground text-sm">
                      {swapMode === 'off'
                        ? t('swaps.no_off_employees', 'ما في موظفين عندهم Off في هذا اليوم')
                        : t('swaps.no_other_shift_employees', 'ما في موظفين بشفت ثاني في هذا اليوم')}
                    </div>
                  ) : (
                    <select
                      className={selectClass}
                      value={targetEmployeeId}
                      onChange={(e) => setTargetEmployeeId(e.target.value)}
                    >
                      <option value="">{t('swaps.select_colleague')}</option>
                      {employees.map((emp: any) => (
                        <option key={emp.id} value={emp.id}>
                          {emp.first_name} {emp.last_name}
                          {swapMode === 'shift' && emp.is_off ? ` (Off)` : ''}
                        </option>
                      ))}
                    </select>
                  )}
                </div>

                {/* Reason */}
                <div className="space-y-2">
                  <Label className="text-muted-foreground ml-1">{t('common.reason')}</Label>
                  <textarea
                    className="w-full h-28 px-4 py-3 rounded-xl bg-muted/50 border-transparent focus:bg-background focus:border-primary text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 resize-none transition-all"
                    placeholder={t('swaps.reason_placeholder')}
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                  />
                </div>
              </div>

              <Button
                className="w-full h-12 rounded-xl text-base font-semibold shadow-lg shadow-primary/20 gap-2"
                onClick={onSubmit}
                disabled={isPending || !canSubmit}
              >
                <Send className="w-5 h-5" /> {t('swaps.send_swap_request')}
              </Button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};
