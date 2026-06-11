import { Check } from 'lucide-react';

import { useI18n } from '../i18n';
import type { RestoreStepKey } from './types';

const steps: RestoreStepKey[] = ['target', 'object', 'destination', 'options', 'progress'];

export function RestoreStepper({ activeStep, maxStep }: { activeStep: RestoreStepKey; maxStep: number }) {
  const { t } = useI18n();
  const activeIndex = steps.indexOf(activeStep);
  return (
    <ol className="restore-stepper" aria-label={t('restoreSteps')}>
      {steps.map((step, index) => {
        const complete = index < activeIndex || index < maxStep;
        const active = index === activeIndex;
        return (
          <li key={step} className={`${active ? 'active' : ''} ${complete ? 'complete' : ''}`} aria-current={active ? 'step' : undefined}>
            <span>{complete ? <Check size={15} /> : index + 1}</span>
            <strong>{t(`restoreStep.${step}`)}</strong>
          </li>
        );
      })}
    </ol>
  );
}

export { steps as restoreSteps };
