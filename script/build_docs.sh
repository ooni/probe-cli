#!/bin/bash
set +ex
DOCS_ROOT=dist/docs/
REPO_NAME="ooni/probe-cli"
COMMIT_HASH=$(git rev-parse --short HEAD)

mkdir -p $DOCS_ROOT

strip_title() {
    # Since the title is already present in the frontmatter, we need to remove
    # it to avoid duplicate titles
    local infile="$1"
    cat $infile | awk 'BEGIN{p=1} /^#/{if(p){p=0; next}} {print}'
}

cat <<EOF>$DOCS_ROOT/00-index.md
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI Probe Engine
description: OONI Probe Engine documentation
slug: probe-engine
---
EOF
strip_title Readme.md >> $DOCS_ROOT/00-index.md
mkdir -p $DOCS_ROOT/img
cp docs/logo.png $DOCS_ROOT/img/
sed -i 's+docs/logo.png+../../../assets/images-probe-engine/logo.png+' $DOCS_ROOT/00-index.md

# design docs
BASE_PATH=docs/design

DOC_PATH=$DOCS_ROOT/00-design.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI Probe Engine Design
description: Design documents for OONI Probe
slug: probe-engine/design
---
EOF
strip_title $BASE_PATH/README.md >> $DOC_PATH

DOC_PATH=$DOCS_ROOT/01-design-oonimkall.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI oonimkall 
description: OONI oonimkall package design documentaton
slug: probe-engine/design/oonimkall
---
EOF
strip_title $BASE_PATH/dd-001-oonimkall.md >> $DOC_PATH

DOC_PATH=$DOCS_ROOT/02-design-netx.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI netx
description: OONI netx package design documentation
slug: probe-engine/design/netx
---
EOF
strip_title $BASE_PATH/dd-002-netx.md >> $DOC_PATH

DOC_PATH=$DOCS_ROOT/03-design-step-by-step.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI step-by-step
description: OONI step-by-step design documentation
slug: probe-engine/design/step-by-step
---
EOF
strip_title $BASE_PATH/dd-003-step-by-step.md >> $DOC_PATH
cp -R $BASE_PATH/img/* $DOCS_ROOT/img/
sed -i 's+img/git-probe-cli-netx-deps.png+../../../assets/images-probe-engine/git-probe-cli-netx-deps.png+' $DOC_PATH
sed -i 's+img/git-probe-cli-change-histogram.png+../../../assets/images-probe-engine/git-probe-cli-change-histogram.png+' $DOC_PATH

DOC_PATH=$DOCS_ROOT/04-design-minioonirunv2.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI minioonirunv2
description: OONI minioonirunv2 design documentation
slug: probe-engine/design/minioonirunv2
---
EOF
strip_title $BASE_PATH/dd-004-minioonirunv2.md >> $DOC_PATH

DOC_PATH=$DOCS_ROOT/05-design-dslx.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI dslx
description: OONI dslx package design documentation
slug: probe-engine/design/dslx
---
EOF
strip_title $BASE_PATH/dd-005-dslx.md >> $DOC_PATH

DOC_PATH=$DOCS_ROOT/06-design-probeservices.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI probeservices
description: OONI probeservices design documentation
slug: probe-engine/design/probeservices
---
EOF
strip_title $BASE_PATH/dd-006-probeservices.md >> $DOC_PATH

DOC_PATH=$DOCS_ROOT/07-design-throttling.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI throttling experiment
description: OONI throttling experiment design documentation
slug: probe-engine/design/throttling
---
EOF
strip_title $BASE_PATH/dd-007-throttling.md >> $DOC_PATH

DOC_PATH=$DOCS_ROOT/08-design-richer-input.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI richer input
description: OONI richer input design documentation
slug: probe-engine/design/richer-input
---
EOF
strip_title $BASE_PATH/dd-008-richer-input.md >> $DOC_PATH

# oonimkall docs
DOC_PATH=$DOCS_ROOT/09-oonimkall.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI oonimkall
description: OONI oonimkall documentation
slug: probe-engine/oonimkall
---
EOF
strip_title pkg/oonimkall/README.md >> $DOC_PATH

# release docs
DOC_PATH=$DOCS_ROOT/10-releasing.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI Probe Release
description: OONI Probe release documentation
slug: probe-engine/releasing
---
EOF
strip_title docs/releasing.md >> $DOC_PATH

# release docs
DOC_PATH=$DOCS_ROOT/10-releasing.md
cat <<EOF>$DOC_PATH
---
# Do not edit! This file is automatically generated
# to edit go to: https://github.com/$REPO_NAME/edit/master/README.md
# version: $REPO_NAME:$COMMIT_HASH
title: OONI Probe Release
description: OONI Probe release documentation
slug: probe-engine/releasing/
---
EOF
strip_title docs/releasing.md >> $DOC_PATH
