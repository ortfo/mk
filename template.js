function URLName(tag) {
  return tag.Plural.replace(/\s|[#%><"]/g, "-")
}

function printf(format, ...args) {
  return format.replace(/%s/g, () => args.shift())
}

function closestTo(target, ...available) {
  // Put the target in its place inside available values
  const a = [...available, target].sort((a, b) => a - b)
  // Get the two closest values to target (if target is not available, target will appear twice, so no problem here)
  const targetIdx = a.indexOf(target)
  const candidates = [a[targetIdx - 1], a[targetIdx + 1]].filter(c => !!c)
  if (candidates.includes(target)) {
    return target
  }
  return candidates.reverse()[0]
}

function lookupTag(name) {
  const tag = all_tags.find(tag =>
    [tag.Plural, tag.Singular, URLName(tag), ...(tag.Aliases || [])]
      .map(s => s.toLowerCase())
      .includes(name.toLowerCase())
  )
  if (tag === undefined) {
    throw Error(
      "No tag found for " + name + ", be sure to add it in tags.yaml."
    )
  }
  return tag
}

function lookupTech(name) {
  const tech = all_technologies.find(tech =>
    [tech.URLName, tech.DisplayName, ...(tech.Aliases || [])]
      .map(s => s.toLowerCase())
      .includes(name.toLowerCase())
  )
  if (tech === undefined) {
    throw Error(
      "No technology found for " +
        name +
        ", be sure to add it in technologies.yaml."
    )
  }
  return tech
}

function asText(html) {
  return html.replace(/<[^>]*>/g, "")
}

function ellipsis(text, maxWords) {
  const words = text.split(" ")
  if (words.length <= maxWords) {
    return text
  }
  return `${words.slice(0, maxWords).join(" ")}…`
}

function Summarize(work, maxWords) {
  try {
    return (
      work.Metadata?.Summary ||
      ellipsis(asText(work.Paragraphs[0].Content), maxWords)
    )
  } catch (error) {
    throw Error(`i can't even ${JSON.stringify(work)}`)
  }
}

function translate_eager(value, context = "") {
  // TODO ...[minify(value)] so that formatting whitespace differences doesn't prevent accessing the value
  return _translations[value + context] || value
}

function translate_context(value, context, ...args) {
  return (
    TRANSLATION_STRING_DELIMITER_OPEN +
    JSON.stringify({ value, args, context }) +
    TRANSLATION_STRING_DELIMITER_CLOSE
  )
}

function translate(value, ...args) {
  return translate_context(value, "", ...args)
}

function AddOctothorpeIfNeeded(value) {
  if (value === "white" || value === "black") {
    return value
  }
  return value.startsWith("#") ? value : `#${value}`
}

function ColorsMap(work) {
  let map = {}
  if (work.Metadata.Colors.Primary != "") {
    map["primary"] = AddOctothorpeIfNeeded(work.Metadata.Colors.Primary)
  }
  if (work.Metadata.Colors.Secondary != "") {
    map["secondary"] = AddOctothorpeIfNeeded(work.Metadata.Colors.Secondary)
  }
  if (work.Metadata.Colors.Tertiary != "") {
    map["tertiary"] = AddOctothorpeIfNeeded(work.Metadata.Colors.Tertiary)
  }
  return map
}

function ColorsCSS(work) {
  return Object.entries(ColorsMap(work))
    .map(([key, value]) => `--${key}:${value}`)
    .join(";")
}

function CreatedAt(work) {
  const unparsed = work.Metadata.Created || work.Metadata.Finished
  return unparsed ? new Date(unparsed) : new Date("0000-11-11")
}

function MostRecentsFirst(works) {
  return works.sort((a, b) => CreatedAt(a) < CreatedAt(b) ? 1 : -1)
}

function IsWIP(work) {
  return (
    work.Metadata.WIP ||
    (work.Metadata.Started != "" && CreatedAt(work).getFullYear() === 0)
  )
}

function thumbnailKey(workOrLayedOutElement) {
  const isLayedOutElement = Object.getOwnPropertyNames(
    workOrLayedOutElement
  ).includes("LayoutIndex")
  if (isLayedOutElement) {
    return workOrLayedOutElement.Path
  }
  if (workOrLayedOutElement?.Metadata?.Thumbnail) {
    return workOrLayedOutElement.Media?.find(
      m => m.Source === workOrLayedOutElement.Metadata.Thumbnail
    )?.Path
  }
  return workOrLayedOutElement.Media?.[0]?.Path
}

function ThumbnailSource(work, resolution, key = null) {
  if (key === null) {
    key = thumbnailKey(work)
  }
  if (!key) {
    return ""
  }
  const availableResolutions = availableResolutionsForThumbnail(work, key)
  if (!availableResolutions.length) {
    throw Error(
      `No thumbnails available for ${key}.\nAvailable thumbnails for work ${
        work.ID
      }: ${work.Media.filter(m => Object.keys(m?.Thumbnails || {})?.length)
        ?.map(m => m.Path)
        .join(", ")}`
    )
  }
  resolution = closestTo(resolution, ...availableResolutions)
  if (resolution > 0) {
    const thumbSource = work.Media.find(m => m.Path === key).Thumbnails[
      resolution
    ]
    if (thumbSource) {
      return media(thumbSource.replace(/^dist\/media\//, ""))
    }
  }
  throw Error(
    `No thumbnail at size ${resolution} for ${key}.\nAvailable resolutions for ${key} (in px): ${availableResolutions.join(
      ", "
    )}`
  )
}

function availableResolutionsForThumbnail(work, key) {
  return Object.keys(
    work.Media.find(m => m.Path === key)?.Thumbnails || {}
  )?.map(parseFloat)
}

function ThumbnailSourcesSet(work, key = null) {
  if (key === null) {
    key = thumbnailKey(work)
  }
  if (!key) return ""

  let result = []

  for (const resolution of availableResolutionsForThumbnail(work, key)) {
    result.push(`${ThumbnailSource(work, resolution, key)} ${resolution}w`)
  }
  return result.join(",")
}

function yearsOfWorks(works) {
  return [...new Set(works.map(w => CreatedAt(w).getFullYear()))]
}

function withTag(works, ...tags) {
  let output = works
  for (let tag of tags) {
    if (typeof tag === "string") {
      tag = lookupTag(tag)
    }
    output = output.filter(work =>
      work.Metadata.Tags.some(t => lookupTag(t)?.Singular === tag.Singular)
    )
  }
  return output
}

function withTech(works, ...techs) {
  let output = works
  for (let tech of techs) {
    if (typeof tech === "string") {
      tech = lookupTech(tech)
    }
    output = output.filter(work =>
      work.Metadata.MadeWith.some(t => lookupTech(t)?.URLName === tech.URLName)
    )
  }
  return output
}

function withWIPStatus(status, works) {
  return works.filter(work => IsWIP(work) === status)
}

function withCreatedYear(works, createdYear) {
  return works.filter(work => {
    const created = CreatedAt(work)
    return created?.getFullYear() === createdYear
  })
}

function excluding(excluseList, works) {
  return works.filter(work => !excluseList.map(w => w.ID).includes(work.ID))
}

function latestWork(works) {
  return works.sort((a, b) => CreatedAt(b) - CreatedAt(a))[0]
}

function finished(works) {
  return works.filter(w => !IsWIP(w))
}

function unfinished(works) {
  return works.filter(IsWIP)
}

const tagged = withTag
const madeWith = withTech
const createdIn = withCreatedYear

// Returns (starting row, ending row, starting column, ending column).
function PositionBounds(l) {
  let startingColumn = Number.MAX_SAFE_INTEGER
  let startingRow = Number.MAX_SAFE_INTEGER
  let endingColumn = 0
  let endingRow = 0

  // printfln("computing grid position for %s", l)
  for (let row of l.Positions) {
    if (row.length != 2) {
      throw Error(
        `A GridArea has an Indices array ${l.Positions} with a row containing ${row.length} != 2 elements`
      )
    }
    if (row[1] < startingColumn) {
      startingColumn = row[1]
    }
    if (row[0] < startingRow) {
      startingRow = row[0]
    }
    if (row[1] > endingColumn) {
      endingColumn = row[1]
    }
    if (row[0] > endingRow) {
      endingRow = row[0]
    }
  }
  return [startingRow, endingRow, startingColumn, endingColumn]
}

// CSS returns CSS statements to declare the position of that element in the content grid.
function CellCSS(l) {
  const [startingRow, endingRow, startingCol, endingCol] = PositionBounds(l)
  return `grid-row: ${startingRow + 1} / ${endingRow + 2}; grid-column: ${
    startingCol + 1
  } / ${endingCol + 2};`
}

function IsColorBright(hexColor) {
  const hex = hexColor.replace(/^#/, "")
  const c_r = parseInt(hex.substr(0, 2), 16)
  const c_g = parseInt(hex.substr(2, 2), 16)
  const c_b = parseInt(hex.substr(4, 2), 16)
  const brightness = (c_r * 299 + c_g * 587 + c_b * 114) / 1000
  return brightness > 155
}

function IsWorkInCollection(work, collection) {
  return (collection.Works || []).map(w => w.ID).includes(work.ID)
}

function CollectionsOfWork(work) {
  return all_collections.filter(c => IsWorkInCollection(work, c))
}
