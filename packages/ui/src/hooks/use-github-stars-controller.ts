"use client";

import * as React from "react";

import { useIsInView, type UseIsInViewOptions } from "./use-is-in-view";

export interface GithubStarsContextType {
  stars: number;
  setStars: (stars: number) => void;
  currentStars: number;
  setCurrentStars: (stars: number) => void;
  isCompleted: boolean;
  isLoading: boolean;
}

export interface UseGithubStarsControllerParams {
  ref?: React.Ref<HTMLDivElement>;
  username?: string;
  repo?: string;
  value?: number;
  delay: number;
  inView: boolean;
  inViewMargin: UseIsInViewOptions["inViewMargin"];
  inViewOnce: boolean;
}

export function useGithubStarsController({
  ref,
  username,
  repo,
  value,
  delay,
  inView,
  inViewMargin,
  inViewOnce,
}: UseGithubStarsControllerParams) {
  const { ref: localRef, isInView } = useIsInView(ref as React.Ref<HTMLDivElement>, {
    inView,
    inViewOnce,
    inViewMargin,
  });

  const [stars, setStars] = React.useState(value ?? 0);
  const [currentStars, setCurrentStars] = React.useState(0);
  const [isLoading, setIsLoading] = React.useState(true);
  const isCompleted = React.useMemo(() => currentStars === stars, [currentStars, stars]);

  React.useEffect(() => {
    if (value !== undefined && username && repo) return;
    if (!isInView) {
      setStars(0);
      setIsLoading(true);
      return;
    }

    const timeout = setTimeout(() => {
      fetch(`https://api.github.com/repos/${username}/${repo}`)
        .then(response => response.json())
        .then(data => {
          if (data && typeof data.stargazers_count === "number") {
            setStars(data.stargazers_count);
          }
        })
        .catch(console.error)
        .finally(() => setIsLoading(false));
    }, delay);

    return () => clearTimeout(timeout);
  }, [username, repo, value, isInView, delay]);

  const contextValue = React.useMemo<GithubStarsContextType>(
    () => ({ stars, currentStars, isCompleted, isLoading, setStars, setCurrentStars }),
    [stars, currentStars, isCompleted, isLoading]
  );

  return { localRef, isLoading, contextValue };
}
