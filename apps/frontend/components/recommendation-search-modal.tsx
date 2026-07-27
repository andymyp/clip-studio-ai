"use client";

import { SparkleIcon } from "@phosphor-icons/react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { ResponsiveModal } from "@/components/ui/responsive-modal";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  trendingSearchSchema,
  type TrendingSearchValues,
} from "@/lib/validations";
import { useRecommendationSearchStore } from "@/stores/recommendation-search-store";

const defaults: TrendingSearchValues = {
  language: "en",
  keywords: "trending",
};

type RecommendationSearchModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSearch: (values: TrendingSearchValues) => void;
  values?: Partial<TrendingSearchValues>;
};

export function RecommendationSearchModal({
  open,
  onOpenChange,
  onSearch,
  values,
}: RecommendationSearchModalProps) {
  const persistedLanguage = useRecommendationSearchStore((state) => state.language);
  const persistedTopic = useRecommendationSearchStore((state) => state.topic);
  const hydrated = useRecommendationSearchStore((state) => state.hydrated);
  const setPersistedSearch = useRecommendationSearchStore((state) => state.setSearch);
  const suppliedLanguage = values?.language;
  const suppliedKeywords = values?.keywords;
  const form = useForm<TrendingSearchValues>({
    resolver: zodResolver(trendingSearchSchema),
    defaultValues: defaults,
  });

  useEffect(() => {
    if (!open) return;
    const supplied = trendingSearchSchema.safeParse({
      language: suppliedLanguage,
      keywords: suppliedKeywords,
    });
    if (supplied.success) {
      form.reset(supplied.data);
      return;
    }
    if (hydrated) {
      form.reset({
        language: persistedLanguage,
        keywords: persistedTopic,
      });
    }
  }, [
    form,
    hydrated,
    open,
    persistedLanguage,
    persistedTopic,
    suppliedKeywords,
    suppliedLanguage,
  ]);

  function submit(formValues: TrendingSearchValues) {
    const normalized = {
      language: formValues.language,
      keywords: formValues.keywords
        .split(",")
        .map((keyword) => keyword.trim())
        .filter(Boolean)
        .join(", "),
    };
    setPersistedSearch(normalized.language, normalized.keywords);
    onOpenChange(false);
    onSearch(normalized);
  }

  return (
    <ResponsiveModal
      open={open}
      onOpenChange={onOpenChange}
      title="Search recommendations"
      description="Choose a language and topic from the recommendation catalog."
    >
      <Form {...form}>
        <form className="space-y-5" onSubmit={form.handleSubmit(submit)}>
          <fieldset disabled={form.formState.isSubmitting} className="space-y-5">
            <FormField
              control={form.control}
              name="language"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Language</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger><SelectValue placeholder="Select language" /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="en">English</SelectItem>
                      <SelectItem value="id">Indonesian</SelectItem>
                      <SelectItem value="es">Spanish</SelectItem>
                      <SelectItem value="pt">Portuguese</SelectItem>
                      <SelectItem value="fr">French</SelectItem>
                      <SelectItem value="de">German</SelectItem>
                      <SelectItem value="ja">Japanese</SelectItem>
                      <SelectItem value="ko">Korean</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="keywords"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Topic</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger><SelectValue placeholder="Select topic" /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {[
                        "trending", "podcast", "interview", "debate", "speech", "documentary",
                        "education", "science", "technology", "history", "business",
                        "startup", "finance", "psychology", "motivation", "health", "story",
                      ].map((topic) => (
                        <SelectItem key={topic} value={topic}>
                          {topic.charAt(0).toUpperCase() + topic.slice(1)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </fieldset>
          <Button type="submit" className="w-full">
            <SparkleIcon />
            Search recommendations
          </Button>
        </form>
      </Form>
    </ResponsiveModal>
  );
}
