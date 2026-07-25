from app.schemas import MarketingMetadata
from app.services.marketing import _normalize


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
