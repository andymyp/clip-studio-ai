from app.schemas import MarketingMetadata
from app.services.marketing import PackagingCandidates, _normalize, _select


def test_marketing_fields_remove_markdown_repetition_and_duplicates() -> None:
    transcript = "Respect men. It's it's that simple. And respect men. It's important."
    result = _normalize(
        MarketingMetadata(
            title="**Respect men. It's it's that simple.**",
            description="Respect men. It's",
            hashtags=["#Shorts", "#Men'sAdvice", "shorts"],
            hook="**Respect men. It's**",
        ),
        transcript,
    )

    assert "*" not in result.title
    assert "It's it's" not in result.title
    assert result.hook != result.title
    assert result.hook.lower() not in transcript.lower()
    assert result.description not in {result.title, result.hook}
    assert len(result.description) >= 35
    assert result.hashtags == ["shorts", "mensadvice"]


def test_editor_selects_distinct_accurate_title_and_opening_hook() -> None:
    transcript = (
        "Most people wait for motivation before they begin. "
        "But action creates the motivation you were waiting for."
    )
    result = _select(
        PackagingCandidates(
            main_topic="motivation through action",
            main_claim="Action creates motivation instead of waiting for it",
            payoff="Starting before feeling ready creates momentum",
            titles=[
                "Why Action Creates Motivation Before You Feel Ready",
                "You Won't Believe This Motivation Secret",
                "Action creates the motivation you were waiting for",
            ],
            hooks=[
                "Stop Waiting To Feel Ready",
                "Why Action Creates Motivation",
                "WATCH UNTIL THE END",
            ],
            description=(
                "A practical explanation of why starting first can generate the drive "
                "needed to keep moving."
            ),
            hashtags=["#motivation", "Productivity", "motivation"],
        ),
        transcript,
    )

    assert result.title == "Why Action Creates Motivation Before You Feel Ready"
    assert result.hook == "Stop Waiting To Feel Ready"
    assert result.title != result.hook
    assert result.hashtags == ["motivation", "productivity"]


def test_invalid_packaging_uses_non_repeating_editorial_fallbacks() -> None:
    transcript = "Respect men. It's it's that simple. And respect men. It's important."
    result = _select(
        PackagingCandidates(
            main_topic="respect",
            main_claim="Respecting men should be straightforward",
            payoff="Respect should be treated as a basic standard",
            titles=["Respect men. It's that simple."] * 3,
            hooks=["Respect men. It's"] * 3,
            description="Respect men. It's",
            hashtags=[],
        ),
        transcript,
    )

    assert result.title == "The Simple Truth About Respect"
    assert result.hook == "Why Respect Actually Matters"
    assert not result.title.startswith("Respect men")
