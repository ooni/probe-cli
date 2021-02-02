package mlablocatev2

import "context"

type ResultRecord resultRecord

func (c Client) Query(ctx context.Context, path string) (ResultRecord, error) {
	out, err := c.query(ctx, path)
	if err != nil {
		return ResultRecord{}, err

	}
	return ResultRecord(out), nil
}

type EntryRecord = entryRecord
