import { createSlice } from '@reduxjs/toolkit';

export type TStep = {
  status: string;
  key?: number;
};

export const initialValue: TPayload = {
  currentStep: 1,
  steps: [
    {
      status: 'progress'
    }
  ],
  lastStep: null,
  hasReachSubmitStep: false,
  testnetSubmitted: false,
  mainnetSubmitted: false
};

const stepperSlice: any = createSlice({
  name: 'stepper',
  initialState: initialValue,
  reducers: {
    setCurrentStep: (state: any, { payload }: any) => {
      state.currentStep = payload.currentStep;
    },
    addStep: (state: any, { payload }: any) => {
      state.steps.push(payload);
    },
    setStepStatus: (state: any, { payload }: any) => {
      state.steps.map((step: any) => {
        if (step.key === payload.step && state.currentStep) {
          step.status = payload.status;
        }
      });
    },
    setHasReachSubmitStep: (state: any, { payload }: any) => {
      state.hasReachSubmitStep = payload.hasReachSubmitStep;
    },
    setLastStep: (state: any, { payload }: any) => {
      state.lastStep = payload.lastStep;
    },
    setStepFormValue: (state: any, { payload }: any) => {
      state.steps.map((step: any) => {
        if (step.key === payload.step && state.currentStep) {
          step.data = payload.formValues;
        }
      });
    },
    getCurrentFormValues: (state: any, { payload }: any | null) => {
      const found = state.steps.filter(
        (step: any) => step.key === payload?.step || state.currentStep
      );
      if (found.length === 1) {
        return found[0].data;
      }
      return null;
    },
    setSubmitStep: (state: any, { payload }: any) => {
      state.hasReachSubmitStep = payload.submitStep;
    },
    // set initial value
    setInitialValue: (state: TPayload, { payload }: any) => {
      state.currentStep = payload.currentStep;
      state.steps = payload.steps;
      state.lastStep = payload.lastStep;
      state.hasReachSubmitStep = payload.hasReachSubmitStep;
      state.testnetSubmitted = payload.testnetSubmitted;
      state.mainnetSubmitted = payload.mainnetSubmitted;
    },
    // get current state
    getCurrentState: (state: TPayload) => {
      return state;
    },
    clearStepper: (state: any) => {
      state.steps = [
        {
          key: 1,
          status: 'progress'
        }
      ];
      state.currentStep = 1;
      state.lastStep = null;
      state.hasReachSubmitStep = false;
      state.testnetSubmitted = false;
      state.mainnetSubmitted = false;
    },
    // set testnet submission
    setTestnetSubmitted: (state: any, { payload }: any) => {
      state.testnetSubmitted = payload.testnetSubmitted;
    },
    // set mainnet submission
    setMainnetSubmitted: (state: any, { payload }: any) => {
      state.mainnetSubmitted = payload.mainnetSubmitted;
    }
  }
});

export const stepperReducer = stepperSlice.reducer;
export const {
  addStep,
  setCurrentStep,
  setStepStatus,
  setLastStep,
  setStepFormValue,
  getCurrentFormValues,
  setSubmitStep,
  clearStepper,
  setHasReachSubmitStep,
  setInitialValue,
  getCurrentState,
  setTestnetSubmitted,
  setMainnetSubmitted
} = stepperSlice.actions;
