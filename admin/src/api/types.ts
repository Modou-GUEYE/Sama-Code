export type Lang = 'fr' | 'wo'
export const LANGS: Lang[] = ['fr', 'wo']

export type Status = 'draft' | 'in_review' | 'published' | 'archived'
export type Role = 'candidate' | 'editor' | 'validator' | 'admin'

export interface Me {
  id: string
  display_name: string
  phone_number: string
  roles: Role[]
}

export interface TokenPair {
  access_token: string
  refresh_token: string
}

export interface MediaAsset {
  id: string
  media_type: 'audio' | 'image'
  mime_type: string
  size_bytes: number
  original_name?: string
  url: string
}

export interface QuestionContent {
  text: string
  audio_media_id: string | null
  image_media_id: string | null
  explanation: string | null
  explanation_audio_media_id: string | null
}

export interface OptionContent {
  text: string
  audio_media_id: string | null
  image_media_id: string | null
}

export interface Option {
  id?: string
  code: string
  is_correct: boolean
  contents: Partial<Record<Lang, OptionContent>>
}

export interface Question {
  id: string
  question_type: 'single' | 'multiple'
  related_lesson_id: string | null
  status: Status
  version: number
  contents: Partial<Record<Lang, QuestionContent>>
  options: Option[]
}

export interface QuestionSummary {
  id: string
  question_type: 'single' | 'multiple'
  status: Status
  version: number
  text_fr: string
  has_wolof: boolean
  option_count: number
  series_count: number
}

export interface License {
  id: string
  code: string
  name: string
  series_count: number
  questions_per_series: number
  max_faults: number
  exam_duration_minutes: number
  is_active: boolean
}

export interface Series {
  id: string
  license_category_id: string
  license_code: string
  series_number: number
  status: Status
  current_version: number
  question_count: number
  questions_per_series: number
  published_at: string | null
}

export interface SeriesQuestion {
  position: number
  question_id: string
  question_type: 'single' | 'multiple'
  status: Status
  text_fr: string
}

export interface SeriesDetail extends Series {
  questions: SeriesQuestion[]
}

export interface Translation {
  title: string
  description?: string | null
  audio_media_id: string | null
}

export type Translations = Partial<Record<Lang, Translation>>

export interface CourseCategory {
  id: string
  code: string
  icon_media_id: string | null
  sort_order: number
  translations: Translations
  course_count: number
}

export interface Lesson {
  id: string
  course_id: string
  status: Status
  sort_order: number
  translations: Translations
  contents?: Record<Lang, Block[]>
}

export interface Course {
  id: string
  category_id: string
  license_category_id: string | null
  status: Status
  sort_order: number
  translations: Translations
  lessons?: Lesson[]
  lesson_count: number
}

export interface Block {
  content_type: 'text' | 'audio' | 'image'
  text: string | null
  media_id: string | null
}

export interface AdminUser {
  id: string
  display_name: string
  phone_number: string
  status: string
  roles: Role[]
  created_at: string
}

export type StatusCounts = Record<Status, number>

export interface Statistics {
  candidates: number
  staff: number
  exam_sessions: number
  questions: StatusCounts
  courses: StatusCounts
  lessons: StatusCounts
  series: StatusCounts
  licenses: { code: string; series_expected: number; series_published: number }[]
}
