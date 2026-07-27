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
  channel_id: string;
  channel_title: string;
  youtube_username: string;
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

export type RecommendationSchedulerStatus = {
  state: "waiting" | "running" | "completed" | "failed";
  catalog_ready: boolean;
  started_at?: string;
  completed_at?: string;
  last_success_at?: string;
  keywords_requested?: string[];
  keywords_succeeded?: string[];
  keyword_errors?: Record<string, string>;
  videos_collected: number;
  error?: string;
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
  youtube_username: string;
  license: string;
  reusable: boolean;
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

export type RenderJob = {
  id: string;
  clip_id: string;
  external_id: string;
  video_title: string;
  thumbnail: string;
  status: "queued" | "pending" | "processing" | "completed" | "failed";
  progress: number;
  message: string;
  score: number;
  duration: number;
  output_path: string;
  media_url: string;
  title: string;
  description: string;
  hashtags: string[];
  error: string;
  hook: string;
  removed_seconds: number;
  pattern_interrupts: number;
  source_username: string;
  playback_speed: number;
  platform_profile: "smart" | "youtube" | "tiktok" | "instagram";
  rights_confirmed: boolean;
  quality_passed: boolean;
  output_width: number;
  output_height: number;
  output_duration: number;
  audio_video_drift: number;
  audio_loudness_lufs: number;
  created_at: string;
  updated_at: string;
};

export type PaginatedResponse<T> = {
  items: T[];
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
};

export type ClipPerformance = {
  id: string;
  render_job_id: string;
  platform: "youtube" | "tiktok" | "instagram";
  views: number;
  likes: number;
  comments: number;
  shares: number;
  average_watch_seconds: number;
  completion_percentage: number;
  engaged_views: number;
  swiped_away_percentage: number;
  replays: number;
  dropoff_second: number;
  viral_score: number;
  created_at: string;
};
