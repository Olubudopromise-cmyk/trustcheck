export type VerifyResponse = {
  input: string;
  type: string;
  status: string;
  trustScore: number;
  summary: string;
};

export type ApiError = {
  error: string;
};
