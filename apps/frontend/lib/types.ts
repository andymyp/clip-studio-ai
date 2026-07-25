export type Video = {
  id?: string;
  title: string;
  url: string;
  thumbnail: string;
  views: number;
  platform: string;
  description?: string;
  duration?: number;
  likes?: number;
  comments?: number;
  transcript?: string;
  created_at?: string;
};

export type VideoSearchResult = {
  external_id: string;
  platform: "youtube" | "reddit";
  category_id: string;
  title: string;
  url: string;
  embed_url: string;
  media_url: string;
  thumbnail: string;
  duration: number;
  views: number;
  likes: number;
  comments: number;
  score: number;
  license: string;
  reusable: boolean;
};

export type GeneratedClip = {
  id: string;
  start: number;
  end: number;
  score: number;
  reason: string;
  output_path: string;
  media_url: string;
};

export type ClipAnalysisJob = {
  id: string;
  user_id: string;
  external_id: string;
  url: string;
  title: string;
  platform: string;
  thumbnail: string;
  status: "queued" | "processing" | "completed" | "failed";
  progress: number;
  message: string;
  clips: GeneratedClip[];
  error: string | null;
};

export type Clip = {
  id: string;
  title: string;
  thumbnail?: string;
  startTime: number;
  endTime: number;
  score: number;
  status: "candidate" | "approved" | "rejected" | "rendered";
};

export type AuthResponse = {
  user: {
    id: string;
    email: string;
    created_at: string;
  };
  tokens: {
    access_token: string;
    refresh_token: string;
    token_type: string;
    expires_in: number;
  };
};

export type TokenResponse = AuthResponse["tokens"];

export type AnalysisJob = {
  id: string;
  video_id: string;
  status: string;
  progress: number;
  message: string;
};

export type JobLog = {
  id: string;
  video_id: string;
  video_title: string;
  status: "queued" | "pending" | "processing" | "completed" | "failed" | "cancelled";
  progress: number;
  message: string;
  created_at: string;
  updated_at: string;
};
