export interface VerificationRequest {
  target: string;
  type: 'domain' | 'email' | 'phone' | 'ip' | 'business';
}

export interface VerificationResult {
  valid: boolean;
  confidence: number;
  message: string;
}
